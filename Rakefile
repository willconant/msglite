require 'rake/clean'

CLEAN.include('*.6')
CLOBBER.include('msglite')

task :default => ["msglite"]

file 'msglite.6' => ['core.go', 'server.go', 'stream.go'] do
	sh "6g -o msglite.6 core.go server.go stream.go"
end

file 'main.6' => ['main.go', 'msglite.6'] do
	sh "6g main.go"
end

file 'msglite' => ['main.6'] do
	sh "6l -o msglite main.6"
end
