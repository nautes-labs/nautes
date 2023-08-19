package testutils

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RandNum() string {
	return fmt.Sprintf("%04d", rand.Intn(999999))
}

func WaitForDelete(client client.Client, obj client.Object) error {
	for i := 0; i < 10; i++ {
		err := client.Delete(context.Background(), obj)
		if apierrors.IsNotFound(err) {
			return nil
		}

		time.Sleep(time.Second)
	}
	return fmt.Errorf("wait for delete %s timeout", obj.GetName())
}

func GenerateNames(format string, len int) []string {
	names := make([]string, len)
	for i := 0; i < len; i++ {
		names[i] = fmt.Sprintf(format, i)
	}
	return names
}
