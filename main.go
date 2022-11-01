package main

import (
	"context"
	"io"
	"net"
	"net/http"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		fx.Provide(
			NewHTTPServer, // アプリケーションにサーバーを提供している
			NewServeMux,   // Routerを設定
			fx.Annotate(
				NewEchoHandler,
				fx.As(new(Route)),
			),
			zap.NewExample, // ロガー
		),
		fx.Invoke(func(*http.Server) {}), // インスタンス化する
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger { // fx自体のログ
			return &fxevent.ZapLogger{Logger: log}
		}),
	).Run()
}

// NewHTTPServer builds an HTTP server that will begin serving requests
// when the Fx application starts.
func NewHTTPServer(lc fx.Lifecycle, mux *http.ServeMux, log *zap.Logger) *http.Server {
	srv := &http.Server{Addr: ":8080", Handler: mux}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			log.Info("Starting HTTP server", zap.String("addr", srv.Addr))
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

// Route is an http.Handler that knows the mux pattern
// under which it will be registered.
// インターフェースを定義
type Route interface {
	http.Handler
	Pattern() string // Pattern reports the path at which this is registered.
}

// EchoHandler is an http.Handler that copies its request body
// back to the response.
type EchoHandler struct {
	log *zap.Logger
}

// NewEchoHandler builds a new EchoHandler.
// Echoハンドラのインスタンスを生成する関数
func NewEchoHandler(log *zap.Logger) *EchoHandler {
	return &EchoHandler{log: log}
}

// ServeHTTP handles an HTTP request to the /echo endpoint.
// EchoHandlerに付与するメソッド  リクエストボディをそのまま返す処理
func (h *EchoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := io.Copy(w, r.Body); err != nil {
		h.log.Warn("Failed to handle request", zap.Error(err))
	}
}

// EchoHandlerにメソッドを追加
func (*EchoHandler) Pattern() string {
	return "/echo"
}

// NewServeMux builds a ServeMux that will route requests
// to the given EchoHandler.
// ハンドラ
func NewServeMux(route Route) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle(route.Pattern(), route)
	return mux
}
