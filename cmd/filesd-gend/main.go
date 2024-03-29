package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/tidwall/buntdb"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	debugMode     bool
	listenAddress string
	sdFilePath    string
	dbPath        string
)

type RegisterType int

const (
	MessageRegister RegisterType = iota
	MessageUnregister
	MessageReplaceTargets
)

type Targets map[uuid.UUID]*TargetGroup

type TargetRegisterMessage struct {
	Action    RegisterType
	TargetId  uuid.UUID
	Target    *TargetGroup
	updatedCh chan bool
}

func generateSd(targets *Targets, targetFile string) error {
	targetFileNameTmp := fmt.Sprintf("%s/.%s.tmp", path.Dir(targetFile), path.Base(targetFile))
	targetFileTmp, err := os.OpenFile(targetFileNameTmp, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}

	defer targetFileTmp.Close()

	targetsList := make([]*TargetGroup, 0)
	for _, target := range *targets {
		targetsList = append(targetsList, target)
	}

	if err := json.NewEncoder(targetFileTmp).Encode(targetsList); err != nil {
		return err
	}

	// Do atomic replace
	return os.Rename(targetFileNameTmp, targetFile)
}

func updateTarget(ctx context.Context, registerCh chan<- *TargetRegisterMessage, target *TargetRegisterMessage) (bool, error) {
	target.updatedCh = make(chan bool)
	registerCh <- target

	select {
	case v := <-target.updatedCh:
		return v, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func main() {
	app := &cli.App{
		Name: "filesd-gend",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "Debug mode (enables debug logging and other goodies)",
				Value:       false,
				EnvVars:     []string{"FILESD_GEND_DEBUG"},
				Destination: &debugMode,
			},
			&cli.StringFlag{
				Name:        "listen-addr",
				Usage:       "HTTP API listen address",
				Value:       ":5555",
				EnvVars:     []string{"FILESD_GEND_HTTP_API_ADDR"},
				Destination: &listenAddress,
			},
			&cli.StringFlag{
				Name:        "sd-file",
				Usage:       "Path to where write Prometheus service discovery file (https://prometheus.io/docs/guides/file-sd/)",
				Value:       "./sd.json",
				EnvVars:     []string{"FILESD_GEND_SD_FILE_PATH"},
				Destination: &sdFilePath,
			},
			&cli.StringFlag{
				Name:        "db",
				Usage:       "Persistent storage for targets (Use ':memory:' for practically no-op)",
				Value:       "./filesd-gend.buntdb",
				EnvVars:     []string{"FILESD_GEND_DB_PATH"},
				Destination: &dbPath,
			},
		},
		Action: entrypoint,
	}
	if err := app.Run(os.Args); err != nil {
		zap.L().Error("encountered unhandled error", zap.Error(err))
	}
}

func entrypoint(cctx *cli.Context) (err error) {
	exitCh := make(chan interface{}, 1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	if err := setupLogging(); err != nil {
		panic(err)
	}
	defer func() { _ = zap.L().Sync() }()

	targetsDb, err := buntdb.Open(dbPath)
	if err != nil {
		zap.L().Panic("failed to open database", zap.Error(err))
	}
	defer targetsDb.Close()

	targetUpdateCh := make(chan *TargetRegisterMessage)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/configure", ConfigureEndpoint(targetUpdateCh))
	mux.HandleFunc("/api/v1/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:         listenAddress,
		Handler:      mux,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go func() {
		<-sigCh
		zap.L().Info("got signal, exiting")
		exitCh <- true
	}()

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			zap.L().Error("failed to listen for http", zap.String("at", srv.Addr), zap.Error(err))
		}
		exitCh <- true
	}()

	targets := make(Targets)

	err = targetsDb.View(func(tx *buntdb.Tx) error {
		return tx.Ascend("", func(key, value string) bool {
			uuidKey := uuid.Must(uuid.FromString(key))
			var decoded TargetGroup
			if err := json.Unmarshal([]byte(value), &decoded); err != nil {
				zap.L().Panic("failed to decode stored target", zap.Error(err))
			}
			zap.L().Debug("loaded previously stored target", zap.String("uuid", uuidKey.String()))
			targets[uuidKey] = &decoded
			return true
		})
	})
	if err != nil {
		zap.L().Panic("failed to load persisted targets", zap.Error(err))
	}

	// Generate sd on startup
	if err := generateSd(&targets, sdFilePath); err != nil {
		zap.L().Error("failed to generate initial sd file", zap.String("path", sdFilePath), zap.Error(err))
	}

	go func() {
		for message := range targetUpdateCh {
			switch message.Action {
			case MessageRegister:
				var duplicateId *uuid.UUID = nil
				for id, other := range targets {
					if other.Eq(message.Target) {
						duplicateId = &id
						break
					}
				}
				if duplicateId != nil {
					zap.L().Debug("attempted to register duplicate, skipped", zap.String("duplicateId", duplicateId.String()))
					message.updatedCh <- false
					continue
				} else {
					zap.L().Debug("registered new target", zap.Reflect("target", message.Target))
				}
				message.updatedCh <- true
				targets[message.TargetId] = message.Target

				// Persist
				err := targetsDb.Update(func(tx *buntdb.Tx) error {
					marshaled, err := json.Marshal(message.Target)
					if err != nil {
						return err
					}
					_, _, err = tx.Set(message.TargetId.String(), string(marshaled), nil)
					return err
				})
				if err != nil {
					zap.L().Error("failed to persist new target", zap.String("uuid", message.TargetId.String()), zap.Error(err))
				}
			case MessageUnregister:
				_, ok := targets[message.TargetId]
				delete(targets, message.TargetId)
				message.updatedCh <- ok

				if ok {
					err := targetsDb.Update(func(tx *buntdb.Tx) error {
						_, err := tx.Delete(message.TargetId.String())
						return err
					})
					if err != nil {
						zap.L().Error("failed to unpersist target", zap.String("uuid", message.TargetId.String()), zap.Error(err))
					}
				} else {
					continue
				}
			case MessageReplaceTargets:
				target, ok := targets[message.TargetId]
				if !ok {
					message.updatedCh <- ok
					continue
				}

				zap.L().Debug("replacing targets", zap.String("uuid", message.TargetId.String()))
				target.Targets = message.Target.Targets
				message.updatedCh <- ok

				err := targetsDb.Update(func(tx *buntdb.Tx) error {
					marshaled, err := json.Marshal(target)
					if err != nil {
						return err
					}
					_, _, err = tx.Set(message.TargetId.String(), string(marshaled), nil)
					return err
				})
				if err != nil {
					zap.L().Error("failed to persist updated target", zap.String("uuid", message.TargetId.String()), zap.Error(err))
				}
			}

			if err := generateSd(&targets, sdFilePath); err != nil {
				zap.L().Error("failed to generate new sd file", zap.String("path", sdFilePath), zap.Error(err))
			} else {
				zap.L().Debug("generated new sd file", zap.String("path", sdFilePath))
			}
		}
	}()

	<-exitCh

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("failed to shut down http server", zap.Error(err))
	}

	if err := srv.Close(); err != nil {
		zap.L().Error("failed to close http server", zap.Error(err))
	}

	return
}

func setupLogging() (err error) {
	var cfg zap.Config
	if debugMode {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
	}

	var logger *zap.Logger
	if logger, err = cfg.Build(); err != nil {
		return
	}

	zap.ReplaceGlobals(logger)
	return
}
