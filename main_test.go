package main

import (
	"errors"
	"testing"
)

func TestAskPass(t *testing.T) {

	someError := errors.New("some error")

	tests := []struct {
		passFromFunc []byte
		errFromFunc  error
		passwordOut  string
		errOut       error
	}{
		{nil, nil, "", nil},
		{[]byte(""), nil, "", nil},
		{[]byte("password1"), nil, "password1", nil},
		{[]byte("password2"), nil, "password2", nil},
		{nil, someError, "", someError},
	}

	for _, test := range tests {
		// update global wrapper
		getPasswdFunc = func() ([]byte, error) {
			return test.passFromFunc, test.errFromFunc
		}

		password, err := askPass()
		if password != test.passwordOut || err != test.errOut {
			t.Errorf("askPass() failed. input: %v, %v. output: %v, %v. expected: %v, %v",
				test.passFromFunc, test.errFromFunc,
				password, err,
				test.passwordOut, test.errOut,
			)
		}
	}
}
