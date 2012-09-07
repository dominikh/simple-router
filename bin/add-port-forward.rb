#!/usr/bin/env ruby
require_relative './port_forward'
require "slop"

# More than likely going to port this to Go later

# NAME
#   add-port-forward -- Add a port forward
#
# DESCRIPTION
#
#   -f, --from
#       Optionally, port forwards can be limited to specific source
#       addresses. Single addresses and netmasks are supported.
#
#   --ports
#       Which ports to forward. Can be a single port (e.g. 80) or a range
#       of ports (e.g. 80-90).
#
#   -p, --protocol
#       Either tcp or udp
#
#   --ttl
#       How long the port forward should exist before getting removed
#       automatically. Specified in minutes. Defaults to unlimited.
#
#   -t, --to
#       The host(s) to forward the port to. Can be a single host (e.g.
#       1.2.3.4) or a range of hosts (e.g. 1.2.3.1-1.2.3.5). Optionally,
#       the ports can be remapped as well, using the same syntax as for
#       --ports (e.g. 1.2.3.4:100-110).
#

opts = Slop.parse do
  banner "./add-port-forward [options]\n"
  on :f, :from=,     required: false
  on     :ports=,    required: true, match: /^\d+(-\d+)?$/
  on :p, :protocol=, required: true, match: /^udp|tcp$/
  on     :ttl=,      required: false, match: /^\d+$/
  on :t, :to=,       required: true
end

opts = opts.to_hash

ports    = opts[:ports]
protocol = opts[:protocol]
ttl      = opts[:ttl]
to       = opts[:to]
from     = opts[:from]

ports = ports.split('-').map(&:to_i)
if ports.any? { |port| port < 0 || port > 65535 }
  $stderr.puts 'Valid ports for --port have to be in the range 0..65535'
  exit 1
end

forward = PortForward.new(ports, protocol, from, to, ttl)
forward.to_iptables(:add).each do |f|
  system f
end

