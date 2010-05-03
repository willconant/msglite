#!/usr/bin/perl

use strict;
use Cwd qw(realpath getcwd);

# MAKE SURE WE'RE IN THE RIGHT PLACE
my $expected = realpath($0);
$expected =~ s/\/[^\/]+$//;

if ($expected ne getcwd()) {
	err("cd to $expected first");
}

my $arch = `uname -a` =~ m/i686/ ? '386' : 'amd64';
my $n = $arch eq '386' ? '8' : '6';
my $compiler = "${n}g";
my $linker = "${n}l";

$ENV{GOARCH} = $arch;

my $target = $ARGV[0] || 'build';

if ($target eq 'build') {
	sys("$compiler -o msglite.$n core.go server.go stream.go http.go");
	sys("$compiler main.go");
	sys("$linker -o msglite main.$n");
	sys("rm *.$n");
}
elsif ($target eq 'clean') {
	sys("rm *.$n") if glob("*.$n");
	sys("rm msglite") if -f 'msglite';
}
else {
	err("invalid target: $target");
}

sub sys {
	print join(' ', @_), "\n";
	system(@_) && exit;
}

sub err {
	print STDERR $_[0], "\n";
	exit 1;
}
