package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"buildestate/backend/internal/access/bootstrap"
)

var errPrivateInput = errors.New("bootstrap requires one JSON object via protected non-terminal standard input; never supply credentials as arguments")

func bootstrapInput(in io.Reader, terminal bool) (bootstrap.Input, error) {
	var input bootstrap.Input
	if terminal {
		return input, errPrivateInput
	}
	// Bound the entire payload before decoding; decoder errors can quote secrets.
	data, err := io.ReadAll(io.LimitReader(in, 16385))
	if err != nil || len(data) > 16384 {
		return input, errPrivateInput
	}
	defer clear(data)
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&input); err != nil {
		return bootstrap.Input{}, errPrivateInput
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return bootstrap.Input{}, errPrivateInput
	}
	return input, nil
}

func runBootstrap(ctx context.Context, db *sql.DB, input bootstrap.Input, out io.Writer) error {
	if err := bootstrap.Create(ctx, db, input); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, "First Administrator and recovery guidance initialized successfully.")
	return err
}

func readBootstrapStdin() (bootstrap.Input, error) {
	info, err := os.Stdin.Stat()
	if err != nil {
		return bootstrap.Input{}, errPrivateInput
	}
	return bootstrapInput(os.Stdin, info.Mode()&os.ModeCharDevice != 0)
}
