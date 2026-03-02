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

    # Install example Caddyfile
    (etc/"caddy-hot-reloader").install "Caddyfile" => "Caddyfile.example" if File.exist?("Caddyfile")
    (etc/"caddy-hot-reloader").install "example.Caddyfile" if File.exist?("example.Caddyfile")

    # Create data directory
    (var/"lib/caddy-hot-reloader").mkpath
  end

  service do
    run [opt_bin/"caddy", "run", "--config", etc/"caddy-hot-reloader/Caddyfile"]
    working_dir var/"lib/caddy-hot-reloader"
    keep_alive true
    log_path var/"log/caddy-hot-reloader.log"
    error_log_path var/"log/caddy-hot-reloader.log"
  end

  def caveats
    <<~EOS
      Caddy with hot-reloader plugin has been installed as 'caddy'.

      To use it, you need to configure a Caddyfile:
        cp #{etc}/caddy-hot-reloader/Caddyfile.example #{etc}/caddy-hot-reloader/Caddyfile
        
      Then edit #{etc}/caddy-hot-reloader/Caddyfile to match your setup.

      To start the service:
        brew services start caddy-hot-reloader

      To test manually:
        caddy run --config #{etc}/caddy-hot-reloader/Caddyfile

      Logs are available at:
        #{var}/log/caddy-hot-reloader.log

      Note: This installs as 'caddy' and will conflict with the official
      Homebrew Caddy formula. If you have both installed, use full paths:
        #{opt_bin}/caddy (this version with hot-reloader)
        /opt/homebrew/bin/caddy (official Caddy, if installed)
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
