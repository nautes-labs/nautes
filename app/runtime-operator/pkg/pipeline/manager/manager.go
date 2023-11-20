package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	hashicorpplugin "github.com/hashicorp/go-plugin"
	"github.com/nautes-labs/nautes/app/runtime-operator/pkg/pipeline/shared"
	nautesconst "github.com/nautes-labs/nautes/pkg/const"
	nautesconfigs "github.com/nautes-labs/nautes/pkg/nautesconfigs"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
)

var logger = logf.Log.WithName("RuntimePluginManagement")

// The PipelinePluginManager can manage all pipeline component plugins.
type PipelinePluginManager interface {
	// GetHookFactory will return a 'HookFactory' that can implement the hook based on the pipeline type and hook type.
	// If the corresponding plugin cannot be found, an error is returned.
	GetHookFactory(pipelineType, hookName string) (shared.HookFactory, error)
}

type RuntimePluginManager struct {
	// HookFactoryIndex is the index to find the 'HookFactory' based on the pipeline type and hook name.
	HookFactoryIndex map[string]map[string]*shared.HookFactory
	// Plugins stores the loaded components, with the index being the file name of the plugin.
	Plugins map[string]pipelinePlugin
	// PluginHomePath is the directory of the plugin to be monitored.
	PluginHomePath string
	// k8sClient is the k8s client of the tenant cluster, which is used to update the plugin information to the tenant cluster.
	k8sClient client.Client
	// waitGroup is used to manage the lifecycle of the plugin management
	// goroutines.
	waitGroup sync.WaitGroup
	config    nautesconfigs.Config
}

var done = make(chan bool, 1)

// NewOptions stores the options of NewPluginManagement.
type NewOptions struct {
	// PluginPath is the path to listen to, which stores all the plugins.
	PluginPath string
	// Client is the k8s client of the tenant cluster.
	Client client.Client
}

const (
	defaultPluginPath = "./plugins"
)

// NewPluginManagement will generate a new PluginManagement, and users can adjust the build parameters through the opts.
func NewPluginManagement(opts *NewOptions) (*RuntimePluginManager, error) {
	var pluginPath string
	var tenantK8sClient client.Client
	if opts != nil {
		pluginPath = opts.PluginPath
		tenantK8sClient = opts.Client
	}

	if pluginPath == "" {
		nautesHomePath, ok := os.LookupEnv(nautesconst.EnvNautesHome)
		if ok {
			pluginPath = filepath.Join(nautesHomePath, defaultPluginPath)
		} else {
			pluginPath = defaultPluginPath
		}
	}

	if tenantK8sClient == nil {
		cl, err := client.New(config.GetConfigOrDie(), client.Options{})
		if err != nil {
			return nil, fmt.Errorf("create kubernetes client failed: %w", err)
		}
		tenantK8sClient = cl
	}

	nautesCFG, err := nautesconfigs.NewNautesConfigFromFile()
	if err != nil {
		return nil, fmt.Errorf("load nautes config failed: %w", err)
	}

	pm := &RuntimePluginManager{
		HookFactoryIndex: map[string]map[string]*shared.HookFactory{},
		Plugins:          map[string]pipelinePlugin{},
		PluginHomePath:   pluginPath,
		k8sClient:        tenantK8sClient,
		waitGroup:        sync.WaitGroup{},
		config:           *nautesCFG,
	}
	return pm, nil
}

// Run will start listening to the plugin directory, and upload the hook information to the cluster where Nautes is located.
func (pm *RuntimePluginManager) Run() error {
	pm.waitGroup.Add(1)
	defer func() {
		for name, pipelinePlugin := range pm.Plugins {
			logger.Info("plugin is terminating", "name", name)
			pipelinePlugin.client.Kill()
		}
		pm.waitGroup.Done()
	}()

	files, err := os.ReadDir(pm.PluginHomePath)
	if err != nil {
		return fmt.Errorf("get plugin failed: %w", err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		err := pm.addPlugin(file.Name())
		if err != nil {
			return err
		}
	}

	var ok bool
	for i := 0; i < 30; i++ {
		err = pm.UploadHooksMetadata()
		if err != nil {
			logger.Error(err, "upload hooks metadata failed")
			time.Sleep(time.Second)
			continue
		}
		ok = true
		break
	}
	if !ok {
		return fmt.Errorf("update metadata failed")
	}

	// Create new watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Start listening for events.
	go pm.watchPlugins(watcher)

	err = watcher.Add(pm.PluginHomePath)
	if err != nil {
		return nil
	}

	// Wait for the execution of the function pm.Kill().
	<-done
	logger.Info("plugin management is terminating")
	return nil
}

// Kill will send an exit message to the Run method and wait for the Run method to exit.
func (pm *RuntimePluginManager) Kill() {
	done <- true
	defer func() {
		pm.waitGroup.Wait()
	}()
}

func (pm *RuntimePluginManager) watchPlugins(watcher *fsnotify.Watcher) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			pluginName := filepath.Base(event.Name)
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
				err := pm.addPlugin(pluginName)
				if err != nil {
					logger.Error(err, "add plugin failed")
				}
				err = pm.UploadHooksMetadata()
				if err != nil {
					logger.Error(err, "upload hooks metadata failed")
				}
			} else if event.Has(fsnotify.Remove) {
				pm.removePlugin(pluginName)
				err := pm.UploadHooksMetadata()
				if err != nil {
					logger.Error(err, "upload hooks metadata failed")
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Error(err, "watch plugin error")
		}
	}
}

// addPlugin will load the plugin based on the passed file name and update the index of 'HookFactory'.
// - If the plugin has already been loaded, it will be deleted before loading.
// - Adding a plugin will fail if the hook name has already been registered by another plugin.
func (pm *RuntimePluginManager) addPlugin(pluginName string) error {
	if _, ok := pm.Plugins[pluginName]; ok {
		pm.removePlugin(pluginName)
	}

	plugin, err := newPlugin(pm.PluginHomePath, pluginName)
	if err != nil {
		return fmt.Errorf("load plugin %s failed: %w", pluginName, err)
	}

	hookSet := sets.New[string]()
	for _, metadata := range plugin.metadataCollection {
		hookSet.Insert(metadata.Name)
	}

	for _, currentPlugin := range pm.Plugins {
		if currentPlugin.pipelineType != plugin.pipelineType {
			continue
		}

		for _, metadata := range currentPlugin.metadataCollection {
			if hookSet.Has(metadata.Name) {
				plugin.client.Kill()
				return fmt.Errorf("hook %s has been registered by plugin %s", metadata.Name, currentPlugin.name)
			}
		}
	}

	pm.Plugins[pluginName] = *plugin
	err = pm.addIndex(pluginName)
	if err != nil {
		return fmt.Errorf("add plugin index %s failed: %w", pluginName, err)
	}
	return nil
}

// removePlugin clears the plugin based on the passed-in file name and deletes the corresponding 'HookFactory' index for the plugin.
func (pm *RuntimePluginManager) removePlugin(pluginName string) {
	pm.removeIndex(pluginName)
	pm.Plugins[pluginName].client.Kill()
	delete(pm.Plugins, pluginName)
}

func (pm *RuntimePluginManager) addIndex(pluginName string) error {
	plugin, ok := pm.Plugins[pluginName]
	if !ok {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	if _, ok := pm.HookFactoryIndex[plugin.pipelineType]; !ok {
		pm.HookFactoryIndex[plugin.pipelineType] = map[string]*shared.HookFactory{}
	}

	for _, metadata := range plugin.metadataCollection {
		pm.HookFactoryIndex[plugin.pipelineType][metadata.Name] = &plugin.factory
	}

	return nil
}

func (pm *RuntimePluginManager) removeIndex(pluginName string) {
	plugin, ok := pm.Plugins[pluginName]
	if !ok {
		return
	}

	if _, ok := pm.HookFactoryIndex[plugin.pipelineType]; !ok {
		return
	}
	for _, metadata := range plugin.metadataCollection {
		delete(pm.HookFactoryIndex[plugin.pipelineType], metadata.Name)
	}
	if len(pm.HookFactoryIndex[plugin.pipelineType]) == 0 {
		delete(pm.HookFactoryIndex, plugin.pipelineType)
	}
}

// UploadHooksMetadata uploads all the metadata information of the loaded plugins to the tenant cluster.
func (pm *RuntimePluginManager) UploadHooksMetadata() error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      shared.ConfigMapNameHooksMetadata,
			Namespace: pm.config.Nautes.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(context.TODO(), pm.k8sClient, cm, func() error {
		data := map[string][]shared.HookMetadata{}
		for pluginName, plugin := range pm.Plugins {
			pipelineType := plugin.pipelineType
			if _, ok := data[pipelineType]; !ok {
				data[pipelineType] = []shared.HookMetadata{}
			}

			data[pipelineType] = append(data[pluginName], plugin.metadataCollection...)
		}

		newData := map[string]string{}
		for pipelineType := range data {
			metadataCollectionStr, err := json.Marshal(data[pipelineType])
			if err != nil {
				return fmt.Errorf("marshal metadata collection failed: %w", err)
			}
			newData[pipelineType] = string(metadataCollectionStr)
		}

		cm.Data = newData
		return nil
	})

	return err
}

func (pm *RuntimePluginManager) GetHookFactory(pipelineType, hookName string) (shared.HookFactory, error) {
	hooks, ok := pm.HookFactoryIndex[pipelineType]
	if !ok {
		return nil, fmt.Errorf("pipeline type %s not found", pipelineType)
	}
	factory, ok := hooks[hookName]
	if !ok {
		return nil, fmt.Errorf("hook %s not found", hookName)
	}
	return *factory, nil
}

// pipelinePlugin is the resource definition of a plugin
type pipelinePlugin struct {
	// name is the file name of the plugin.
	name string
	// client is an instance of hashicorp's plugin client.
	client *hashicorpplugin.Client
	// factory is the interface implementation obtained from the plugin client.
	factory shared.HookFactory
	// pipelineType is the plugin type supported by this plugin.
	pipelineType string
	// metadataCollection is the metadata information of the hooks supported by the plugin.
	metadataCollection []shared.HookMetadata
}

func newPlugin(path, pluginName string) (*pipelinePlugin, error) {
	pluginFullPath := filepath.Join(path, pluginName)
	pluginClient := hashicorpplugin.NewClient(&hashicorpplugin.ClientConfig{
		HandshakeConfig:  shared.Handshake,
		Plugins:          shared.PluginMap,
		Cmd:              exec.Command(pluginFullPath),
		AllowedProtocols: []hashicorpplugin.Protocol{hashicorpplugin.ProtocolGRPC},
	})
	grpcClient, err := pluginClient.Client()
	if err != nil {
		return nil, err
	}
	raw, err := grpcClient.Dispense(shared.PluginTypeGRPC)
	if err != nil {
		return nil, err
	}
	hookFactory, ok := raw.(shared.HookFactory)
	if !ok {
		return nil, fmt.Errorf("%s is not a pipeline plugin", pluginName)
	}

	pipelineType, err := hookFactory.GetPipelineType()
	if err != nil {
		return nil, fmt.Errorf("get pipeline type failed: %w", err)
	}

	metadataCollection, err := hookFactory.GetHooksMetadata()
	if err != nil {
		return nil, fmt.Errorf("get hooks metadata failed: %w", err)
	}

	return &pipelinePlugin{
		name:               pluginName,
		client:             pluginClient,
		factory:            hookFactory,
		pipelineType:       pipelineType,
		metadataCollection: metadataCollection,
	}, nil
}
