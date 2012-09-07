#!/usr/bin/env ruby

def hostname_via_dhcp(ip)
  IO.popen("omshell", "w+") do |pipe|
    pipe.puts 'port 9991'
    pipe.puts 'key omapi_key "dl0Af3EePO2kubmSZ1P9AoIBfhYGCe9B6vtgx1hByFpRKOjGChytYrRvjX2D764QxFJoorzJLOnUyzcFW4Wrpg=="'
    pipe.puts 'connect'
    pipe.puts 'new "lease"'
    pipe.puts 'set ip-address = ' + ARGV[0]
    pipe.puts 'open'
    pipe.close_write

    lines = pipe.readlines
    lines.each do |line|
      if line =~ /^client-hostname = "(.+)"$/
        return $1
      end
    end
  end

  return nil
end

def hostname_via_rdns(ip)
  output = `host '#{ip}'`
  if $? == 0
    return output.split(" ").last
  end

  return nil
end

ip = ARGV[0]
puts hostname_via_dhcp(ip) || hostname_via_rdns(ip) || ip
