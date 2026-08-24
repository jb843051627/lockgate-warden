package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jb843051627/lockgate-warden/internal/model"
)

// respond 统一响应封装：2xx JSON / 错误哨兵映射。
func respond(w http.ResponseWriter, v any, err error) {
	if err != nil {
		switch {
		case errors.Is(err, model.ErrNotFound):
			writeErr(w, http.StatusNotFound, err)
		case errors.Is(err, model.ErrConflict),
			errors.Is(err, model.ErrAckRequired),
			errors.Is(err, model.ErrOrphanSensor),
			errors.Is(err, model.ErrDisabledSensor),
			errors.Is(err, model.ErrChecksum),
			errors.Is(err, model.ErrBadWindow),
			errors.Is(err, model.ErrFutureWindow),
			errors.Is(err, model.ErrExpiredWindow),
			errors.Is(err, model.ErrEmptyBatch):
			writeErr(w, http.StatusUnprocessableEntity, err)
		default:
			writeErr(w, http.StatusBadRequest, err)
		}
		return
	}
	if v == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeOK(w, v)
}

func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func queryInt(r *http.Request, key string, def int64) int64 {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return def
	}
	var n int64
	for _, c := range raw {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int64(c-'0')
	}
	return n
}

func queryString(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}
