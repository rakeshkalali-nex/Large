// Package transport is shared by route modules without importing the composition root.
package transport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

const MaxJSONBody int64 = 1 << 20
type requestIDKey struct{}
func WithRequestID(ctx context.Context,id string) context.Context {return context.WithValue(ctx,requestIDKey{},id)}
func RequestID(ctx context.Context) string { id,_:=ctx.Value(requestIDKey{}).(string);return id }
func JSON(w http.ResponseWriter,r *http.Request,status int,data any) {
	w.Header().Set("Content-Type","application/json; charset=utf-8")
	w.Header().Set("Cache-Control","no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data":data,"requestId":RequestID(r.Context())})
}
func Error(w http.ResponseWriter,r *http.Request,status int,code,message string) {
	w.Header().Set("Content-Type","application/json; charset=utf-8")
	w.Header().Set("Cache-Control","no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error":map[string]string{"code":code,"message":message},"requestId":RequestID(r.Context())})
}
func Unavailable(w http.ResponseWriter,r *http.Request) { Error(w,r,503,"SERVICE_UNAVAILABLE","This capability is not available.") }

// DecodeJSON writes a safe contract error on failure and accepts one strict JSON value.
func DecodeJSON(w http.ResponseWriter,r *http.Request,dst any) bool {
	media,_,err:=mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err!=nil || media!="application/json" {Error(w,r,415,"UNSUPPORTED_MEDIA_TYPE","Expected application/json.");return false}
	r.Body=http.MaxBytesReader(w,r.Body,MaxJSONBody)
	d:=json.NewDecoder(r.Body);d.DisallowUnknownFields()
	err=d.Decode(dst)
	if err==nil {var extra any; if e:=d.Decode(&extra);e!=io.EOF {if e==nil {e=errors.New("extra JSON value")};err=e}}
	if err!=nil {var limit *http.MaxBytesError;if errors.As(err,&limit) {Error(w,r,413,"BODY_TOO_LARGE","Request body is too large.")} else {Error(w,r,400,"MALFORMED_REQUEST","Request body is invalid.")};return false}
	return true
}
