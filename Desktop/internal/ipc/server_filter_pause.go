package ipc

import (
	"encoding/json"
	"io"
	"net/http"
)

// handleFilterPause 处理 GET / POST /api/v1/filter/pause
//
// 借鉴 Technitium TemporaryDisableBlocking / pi-hole piholeEnable 的定时暂停语义：
// 暂停为运行时状态，重启服务即恢复拦截。
func (s *Server) handleFilterPause(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		res, err := s.controller.GetFilterPause(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, Response[any]{Success: false, Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, Response[*FilterPauseResponse]{Success: true, Data: res})
		return
	}

	if r.Method == http.MethodPost {
		body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Response[any]{Success: false, Error: "读取请求体失败: " + err.Error()})
			return
		}
		var req FilterPauseRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, Response[any]{Success: false, Error: "解析暂停请求失败: " + err.Error()})
			return
		}
		if err := s.controller.SetFilterPause(r.Context(), req); err != nil {
			writeJSON(w, http.StatusInternalServerError, Response[any]{Success: false, Error: err.Error()})
			return
		}
		msg := "拦截已恢复"
		if req.Paused {
			msg = "拦截已暂停"
		}
		writeJSON(w, http.StatusOK, Response[string]{Success: true, Message: msg})
		return
	}

	writeJSON(w, http.StatusMethodNotAllowed, Response[any]{Success: false, Error: "只支持 GET / POST 请求"})
}
