package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// wait there until SIGINT or SIGTERM call
		s := <-quit

		app.logger.PrintInfo("shutting down server", map[string]string{
			"signal": s.String(),
		})

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			// if err != nil then the application is terminated ?
			// or something wrong then can't terminate the application ?
			shutdownError <- err
		}

		app.logger.PrintInfo("completing background task", map[string]string{
			"addr": srv.Addr,
		})

		app.wg.Wait()
		shutdownError <- nil

	}()

	app.logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
		"env":  app.config.env,
	})

	// calling shutdown will case ListenAndServe() to immediately return a http.ErrServerClosed error.
	// only return err that not http.ErrServerClosed error to specially checking
	// err := srv.ListenAndServe()

	err := srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError

	// greace full shutdown is NOT OK
	if err != nil {
		return err
	}

	app.logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})

	return nil
}
