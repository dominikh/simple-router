#!/usr/bin/env ruby

last_bytes_in = Hash.new(0)
last_bytes_out = Hash.new(0)
last_time = nil


while true
  this_time = Time.now

  # entries_total = `sudo /sbin/iptables -L FORWARD -v -n -x`.lines.to_a[2..-1]
  entries_in    = `sudo /sbin/iptables -L TRAFFIC_IN -v -n -x`.lines.to_a[2..-1]
  entries_out   = `sudo /sbin/iptables -L TRAFFIC_OUT -v -n -x`.lines.to_a[2..-1]

  total_in = 0
  total_out = 0
  total_per_second_in = 0
  total_per_second_out = 0

  puts "START"

  entries_in.zip(entries_out) do |entry_in, entry_out|
    _, bytes_in, _, _, _, _, _, _, destination = entry_in.split(" ")
    _, bytes_out, *_ = entry_out.split(" ")
    bytes_in = bytes_in.to_i
    bytes_out = bytes_out.to_i
    destination.chomp!

    unless last_time.nil?
      if bytes_in > 0 || bytes_out > 0
        bytes_per_second_in = (bytes_in - last_bytes_in[destination]) / (this_time - last_time)
        bytes_per_second_out = (bytes_out - last_bytes_out[destination]) / (this_time - last_time)

        total_per_second_in += bytes_per_second_in
        total_per_second_out += bytes_per_second_out

        puts "%f;%s;%d;%d;%d;%d" % [
                                         this_time.to_f,
                                         destination,
                                         bytes_per_second_in,
                                         bytes_per_second_out,
                                         bytes_in,
                                         bytes_out
                                        ]
      end
    end

    last_bytes_in[destination] = bytes_in
    last_bytes_out[destination] = bytes_out

    total_in += bytes_in
    total_out += bytes_out
  end
  puts "%f;total;%d;%d;%d;%d" % [this_time.to_f, total_per_second_in, total_per_second_out, total_in, total_out]
  puts "END"

  last_time = this_time
  sleep 0.5
end


