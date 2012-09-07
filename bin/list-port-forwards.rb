#!/usr/bin/env ruby
require_relative './port_forward'

header  = "Ports\tProtocol\tFrom\tTo\tRemove command"
entries = `sudo /sbin/iptables -t nat -L PORTFORWARDS_DNAT -n`.lines.to_a[2..-1] || []

entries.map! do |entry|
  # TODO support for ttl
  _, protocol, _, source, _, _, ports, to = entry.split(' ')

  ports = ports.split(':', 2).last
  to    = to.split(':',    2).last

  if source != '0.0.0.0/0'
    remove = "remove-port-forward --ports '%s' -p '%s' -f '%s' -t '%s'" % [ports, protocol, source, to]
  else
    remove = "remove-port-forward --ports '%s' -p '%s' -t '%s'" % [ports, protocol, to]
    source = 'any'
  end

  "%s\t%s\t%s\t%s\t%s" % [ports, protocol, source, to, remove]
end

all = [header, *entries].join("\n")

output = `echo "#{all}" | column -s "\t" -t`.lines.to_a
print "\e[1m#{output[0]}\e[0m"
puts output[1..-1]
