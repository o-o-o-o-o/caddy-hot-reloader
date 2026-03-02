# typed: false
# frozen_string_literal: true

# Caddy with hot-reloader plugin for wildcard local development
class CaddyHotReloader < Formula
  desc "Caddy web server with hot-reload plugin for wildcard local development"
  homepage "https://github.com/o-o-o-o-o/caddy-hot-reloader"
  url "https://github.com/o-o-o-o-o/caddy-hot-reloader/archive/refs/tags/v0.5.0.tar.gz"
  sha256 "9f404259abd1be315154f3f706aad42492ff0a7056e4b439d7994cf4c5f56815"
  license "Apache-2.0"
  head "https://github.com/o-o-o-o-o/caddy-hot-reloader.git", branch: "main"

  depends_on "go" => :build

  def install
    # Build custom Caddy with hot-reloader plugin using xcaddy
    system "go", "run", "github.com/caddyserver/xcaddy/cmd/xcaddy@latest", "build",
           "--with", "github.com/o-o-o-o-o/caddy-hot-reloader=#{buildpath}",
           "--output", "#{bin}/caddy"

    # Create wrapper script to pass config path to service
    wrapper = bin/"caddy-wrapper"
    wrapper.write <<~WRAPPER
      #!/bin/bash
      exec "#{opt_bin}/caddy" run --config "#{HOMEBREW_PREFIX}/etc/Caddyfile"
    WRAPPER
    wrapper.chmod 0o755

    # Install example Caddyfile to /opt/homebrew/etc (not in formula subdirectory)
    (HOMEBREW_PREFIX/"etc").install "Caddyfile" => "Caddyfile.example" if File.exist?("Caddyfile")
    (HOMEBREW_PREFIX/"etc").install "example.Caddyfile" => "Caddyfile.example" if File.exist?("example.Caddyfile") && !File.exist?("Caddyfile")

    # Create data directory
    (var/"lib/caddy-hot-reloader").mkpath
  end

  service do
    run opt_bin/"caddy-wrapper"
    working_dir var/"lib/caddy-hot-reloader"
    keep_alive true
    log_path var/"log/caddy-hot-reloader.log"
    error_log_path var/"log/caddy-hot-reloader.log"
  end

  def caveats
    <<~EOS
      Caddy with hot-reloader plugin has been installed as 'caddy'.

      QUICK START:
        cp #{etc}/Caddyfile.example #{etc}/Caddyfile
        brew services start caddy-hot-reloader

      CUSTOM CADDYFILE LOCATION (2 options):

      Option 1: Symlink your Caddyfile to the default location:
        ln -sf /path/to/your/Caddyfile #{etc}/Caddyfile
        brew services restart caddy-hot-reloader

      Option 2: Run manually with custom path (without service):
        #{opt_bin}/caddy run --config /path/to/your/Caddyfile

      AVAILABLE COMMANDS:
        Start service:  brew services start caddy-hot-reloader
        Stop service:   brew services stop caddy-hot-reloader
        Restart:        brew services restart caddy-hot-reloader
        Logs:           #{var}/log/caddy-hot-reloader.log

      CONFLICT NOTE:
      This installs as 'caddy' and may conflict with official Homebrew Caddy.
      Use full path if both are installed: #{opt_bin}/caddy

      For more info visit: https://github.com/o-o-o-o-o/caddy-hot-reloader
    EOS
  end

  test do
    output = shell_output("#{bin}/caddy version")
    assert_match "v2", output
    
    # Test that the hot_reloader module is available
    (testpath/"Caddyfile").write <<~EOS
      {
        admin off
      }
      :8080 {
        hot_reloader
        respond "OK"
      }
    EOS
    
    system bin/"caddy", "adapt", "--config", testpath/"Caddyfile"
  end
end
