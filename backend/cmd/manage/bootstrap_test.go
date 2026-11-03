package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"buildestate/backend/internal/access/bootstrap"
	"buildestate/backend/internal/testsupport"
)

func TestBootstrapInputRejectsUnsafeAndMalformedInput(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		terminal   bool
	}{
		{"terminal", `{"password":"secret-phrase"}`, true},
		{"malformed", `{"password":"secret-phrase"`, false},
		{"unknown", `{"secret-phrase":"value"}`, false},
		{"multiple", `{} {"password":"secret-phrase"}`, false},
		{"oversize", strings.Repeat("secret-phrase", 2000), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := bootstrapInput(strings.NewReader(tc.body), tc.terminal)
			if !errors.Is(err, errPrivateInput) || strings.Contains(err.Error(), "secret-phrase") {
				t.Fatalf("unsafe input error: %v", err)
			}
		})
	}
}

func TestBootstrapCommandSecrecyAndRepeat(t *testing.T) {
	in := bootstrap.Input{DisplayName: "Operator", Email: "operator@example.invalid", Password: "private command phrase", CompanyName: "Company", RecoveryContactLabel: "Help desk", RecoveryInstructions: "Contact the help desk in person."}
	payload, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := bootstrapInput(bytes.NewReader(payload), false)
	if err != nil {
		t.Fatal(err)
	}
	db := testsupport.Database(t)
	var out bytes.Buffer
	if err = runBootstrap(context.Background(), db, decoded, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), in.Password) || strings.Contains(out.String(), in.Email) || strings.Contains(out.String(), "argon2") {
		t.Fatal("private value echoed")
	}
	out.Reset()
	if err = runBootstrap(context.Background(), db, decoded, &out); !errors.Is(err, bootstrap.ErrExists) || out.Len() != 0 {
		t.Fatalf("repeat result: %v", err)
	}
}

func TestCredentialArgumentsRejectedWithoutEcho(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = []string{"manage", "bootstrap", "private argument phrase"}
	err := run()
	if err == nil || strings.Contains(err.Error(), "private argument phrase") {
		t.Fatal("credential argument accepted or echoed")
	}
}
