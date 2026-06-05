# Get version from git tag (e.g., "v1.0.0" or "v1.0.0-3-g1a2b3c4")
def git_version
  version = `git describe --tags --always --match 'v*' 2>/dev/null`.strip
  version.empty? ? "dev" : version.sub(/^v/, "")
end

@pdfa11y_version = git_version

desc "Show rake description"
task :default do
  puts
  puts "Run 'rake -T' for a list of tasks."
  puts
  puts "Use 'rake build' to build the 'pdfa11y' binary."
  puts
end

desc "Build the pdfa11y binary into ./pdfa11y"
task :build do
  sh "go build -ldflags '-s -w -X main.Version=#{@pdfa11y_version}' -o pdfa11y ./cmd/pdfa11y"
end

desc "Install pdfa11y to $GOBIN (or $GOPATH/bin)"
task :install do
  sh "go install -ldflags '-s -w -X main.Version=#{@pdfa11y_version}' ./cmd/pdfa11y"
end

desc "Show version information"
task :showversion do
  puts "pdfa11y version #{@pdfa11y_version}"
end
