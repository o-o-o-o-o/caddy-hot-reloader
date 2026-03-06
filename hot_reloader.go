func (rw *responseWrapper) Flush() {
    if rw.shouldInject && len(rw.buffer) > 0 {
        body := string(rw.buffer)
        if idx := strings.LastIndex(body, "</body>"); idx != -1 {
            injected := body[:idx] + clientScript + body[idx:]
            rw.hotReloader.logger.Info("injected hot reload script", zap.String("host", rw.host), zap.Int("original_size", len(rw.buffer)), zap.Int("injected_size", len(injected)))
            rw.ResponseWriter.WriteHeader(rw.statusCode)
            rw.ResponseWriter.Write([]byte(injected))
        } else {
            rw.hotReloader.logger.Warn("no </body> tag found, appending diagnostic script", zap.String("host", rw.host), zap.Int("size", len(rw.buffer)))
            diagnosticScript := `<script>console.error("[Caddy Hot Reload] Failed to inject script: no closing </body> tag found. Check server HTML output.");</script>`
            injected := string(rw.buffer) + diagnosticScript
            rw.ResponseWriter.WriteHeader(rw.statusCode)
            rw.ResponseWriter.Write([]byte(injected))
        }
    }
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
    var hr HotReloader
    err := hr.UnmarshalCaddyfile(h.Dispenser)
    return &hr, err
}