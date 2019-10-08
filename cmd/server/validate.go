package main

import (
	"context"

	"firebase.google.com/go/auth"
)

func (app *booksingApp) validateToken(idToken string) (*auth.Token, error) {

	c := context.Background()

	token, err := app.authClient.VerifyIDToken(c, idToken)
	if err != nil {
		return nil, err
	}

	return token, nil
}
