# typed: false
# frozen_string_literal: true

# Caddy with hot-reloader plugin for wildcard local development
class CaddyHotReloader < Formula
  desc "Caddy web server with hot-reload plugin for wildcard local development"
  homepage "https://github.com/o-o-o-o-o/caddy-hot-reloader"
  url "https://github.com/o-o-o-o-o/caddy-hot-reloader/archive/refs/tags/v0.6.4.tar.gz"
  sha256 "8026b181317ade2489b5a29ab212abf1d7a072780b371d6ffa6218e4cee59a70"
  license "Apache-2.0"
  head "https://github.com/o-o-o-o-o/caddy-hot-reloader.git", branch: "main"

  depends_on "go" => :build

  def install
    # Build custom Caddy with hot-reloader plugin using pinned xcaddy.
    # Pinning avoids non-reproducible builds caused by upstream latest changes.
    system "go", "run", "github.com/caddyserver/xcaddy/cmd/xcaddy@v0.4.2", "build",
           "--with", "github.com/o-o-o-o-o/caddy-hot-reloader=#{buildpath}",
           "--output", "#{bin}/caddy"

    # Install example Caddyfile to /opt/homebrew/etc (not in formula subdirectory)
    (HOMEBREW_PREFIX/"etc").install "Caddyfile" => "Caddyfile.example" if File.exist?("Caddyfile")
    if File.exist?("example.Caddyfile") && !File.exist?("Caddyfile")
      (HOMEBREW_PREFIX/"etc").install "example.Caddyfile" => "Caddyfile.example"
    end

    # Create data and log directories for service startup
    (var/"lib/caddy-hot-reloader").mkpath
    (var/"log").mkpath
  end

  service do
    run [opt_bin/"caddy", "run", "--config", "#{HOMEBREW_PREFIX}/etc/Caddyfile"]
    working_dir var/"lib/caddy-hot-reloader"
    keep_alive true
    log_path var/"log/caddy-hot-reloader.log"
    error_log_path var/"log/caddy-hot-reloader.log"
  end

  def caveats
    <<~EOS
      Caddy with hot-reloader plugin has been installed as 'caddy'.

      ⚠️  IMPORTANT FOR BOOT STARTUP:
      Before using 'brew services start', ensure a Caddyfile exists at:
        #{HOMEBREW_PREFIX}/etc/Caddyfile
      If missing, the service will fail at boot. Copy the example:
        cp #{etc}/Caddyfile.example #{etc}/Caddyfile
        nano #{etc}/Caddyfile  # Edit with your configuration

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
        Logs:           tail -f #{var}/log/caddy-hot-reloader.log
        Status:         brew services list

      TROUBLESHOOTING:
      If service fails at boot but works manually:
        1. Check logs: tail -f #{var}/log/caddy-hot-reloader.log
        2. Verify Caddyfile exists: ls -l #{etc}/Caddyfile
        3. Reinstall service: brew services stop caddy-hot-reloader && brew services start caddy-hot-reloader

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
