package flycheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	su "github.com/fly-apps/postgres-standalone/pkg/checks"
)

const Port = 5500

func StartCheckListener() {
	http.HandleFunc("/flycheck/vm", runVMChecks)
	http.HandleFunc("/flycheck/pg", runPGChecks)

	fmt.Printf("Listening on port %d", Port)
	http.ListenAndServe(fmt.Sprintf(":%d", Port), nil)
}

func runVMChecks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (5 * time.Second))
	defer cancel()
	suite := &su.CheckSuite{Name: "VM"}
	suite = CheckVM(suite)

	go func(ctx context.Context) {
		suite.Process(ctx)
		cancel()
	}(ctx)

	select {
	case <-ctx.Done():
		handleCheckResponse(w, suite, false)
	}
}

func runPGChecks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (5 * time.Second))
	defer cancel()
	suite := &su.CheckSuite{Name: "PG"}
	suite, err := CheckPostgreSQL(ctx, suite)
	if err != nil {
		suite.ErrOnSetup = err
		cancel()
	}

	go func() {
		suite.Process(ctx)
		cancel()
	}()

	select {
	case <-ctx.Done():
		handleCheckResponse(w, suite, false)
	}
}

func handleCheckResponse(w http.ResponseWriter, suite *su.CheckSuite, raw bool) {
	if suite.ErrOnSetup != nil {
		handleError(w, suite.ErrOnSetup)
		return
	}
	var result string
	if raw {
		result = suite.RawResult()
	} else {
		result = suite.Result()
	}
	if !suite.Passed() {
		handleError(w, fmt.Errorf(result))
		return
	}
	json.NewEncoder(w).Encode(result)
}

func handleError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(err.Error())
}
