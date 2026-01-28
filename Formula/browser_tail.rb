# Homebrew formula for browser_tail
# To use: brew install ajsharma/tap/browser_tail

class BrowserTail < Formula
  desc "Real-time Chrome DevTools event logger for debugging and AI agents"
  homepage "https://github.com/ajsharma/browser_tail"
  license "MIT"
  version "0.1.0"

  on_macos do
    on_intel do
      url "https://github.com/ajsharma/browser_tail/releases/download/v#{version}/browser_tail_#{version}_darwin_amd64.tar.gz"
      # sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
    on_arm do
      url "https://github.com/ajsharma/browser_tail/releases/download/v#{version}/browser_tail_#{version}_darwin_arm64.tar.gz"
      # sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/ajsharma/browser_tail/releases/download/v#{version}/browser_tail_#{version}_linux_amd64.tar.gz"
      # sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
    on_arm do
      url "https://github.com/ajsharma/browser_tail/releases/download/v#{version}/browser_tail_#{version}_linux_arm64.tar.gz"
      # sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
  end

  def install
    bin.install "browser_tail"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/browser_tail --version")
  end
end
