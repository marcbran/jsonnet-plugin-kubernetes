package kubernetes

import (
	"context"
	"errors"
	"net/http"

	"github.com/marcbran/jpoet/pkg/jpoet"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func invalidArgs(err error) error {
	return jpoet.NewFailure(http.StatusBadRequest, err.Error())
}

func failure(err error) error {
	if err == nil || errors.Is(err, context.Canceled) {
		return err
	}
	var apiStatus apierrors.APIStatus
	if errors.As(err, &apiStatus) {
		s := apiStatus.Status()
		return &jpoet.Failure{Code: int(s.Code), Reason: string(s.Reason), Message: err.Error()}
	}
	return jpoet.NewFailure(http.StatusInternalServerError, err.Error())
}
