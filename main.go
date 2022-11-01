package main

import (
	"context"
	"fmt"
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
			fx.Annotate(
				NewServeMux,
				fx.ParamTags(`group:"routes"`),
			),
			AsRoute(NewEchoHandler), // AsRouteでハンドラをラップしている
			AsRoute(NewHelloHandler),
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

// HelloHandler is an HTTP handler that
// prints a greeting to the user.
// 新たに作成したハンドラ Helloと返す
type HelloHandler struct {
	log *zap.Logger
}

// NewEchoHandler builds a new EchoHandler.
// Echoハンドラのインスタンスを生成する関数
func NewEchoHandler(log *zap.Logger) *EchoHandler {
	return &EchoHandler{log: log}
}

// NewHelloHandler builds a new HelloHandler.
// HelloHandlerインスタンスを生成する
func NewHelloHandler(log *zap.Logger) *HelloHandler {
	return &HelloHandler{log: log}
}

// ServeHTTP handles an HTTP request to the /echo endpoint.
// EchoHandlerに付与するメソッド  リクエストボディをそのまま返す処理
func (h *EchoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := io.Copy(w, r.Body); err != nil {
		h.log.Warn("Failed to handle request", zap.Error(err))
	}
}

// HelloHandlerに付与するメソッド  リクエストボディにHelloを付けて返す
func (h *HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("Failed to read request", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if _, err := fmt.Fprintf(w, "Hello, %s\n", body); err != nil {
		h.log.Error("Failed to write response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// EchoHandlerにPattern()メソッドを追加
func (*EchoHandler) Pattern() string {
	return "/echo"
}

// HelloHandlerにPattern()メソッドを追加
func (*HelloHandler) Pattern() string {
	return "/hello"
}

// AsRoute annotates the given constructor to state that
// it provides a route to the "routes" group.
// ハンドラを入力して、fx.Annotate()を出力する
func AsRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Route)),
		fx.ResultTags(`group:"routes"`),
	)
}

// NewServeMux builds a ServeMux that will route requests
// to the given EchoHandler.
// ハンドラ
func NewServeMux(routes []Route) *http.ServeMux {
	mux := http.NewServeMux()
	for _, route := range routes {
		mux.Handle(route.Pattern(), route)
	}
	return mux
}
