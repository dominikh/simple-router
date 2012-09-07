#!/usr/bin/env ruby
require "filesize"

class Numeric
  def round_up(nearest)
    self % nearest == 0 ? self : self + nearest - (self % nearest)
  end

  def round_down(nearest)
    self % nearest == 0 ? self : self - (self % nearest)
  end

  def round_nearest(nearest)
    if self % nearest >= nearest/2.0
      self.round_up(nearest)
    else
      self.round_down(nearest)
    end
  end
end


MAX_IN = 13107200
MAX_OUT = 655360

COLORS = [84, 83, 82, 46, 40, 226, 220, 214, 208, 202, 196]

header = "Host\t         Down\t           Up\tTotal down\t  Total up\n"

hostnames = {}
IO.popen("~/bin/traffic-stats-raw", "r") do |pipe|
  while line = pipe.gets
    line.chomp!
    if line == "START"
      output = []
      colors = []
      next
    end

    if line == "END"
      s = ""
      lines = `echo "#{([header] + output).join("\n")}" | column -s "\t" -t`.lines.to_a
      s << "\033[2J\033[;H"
      s <<  "\e[1m#{lines.first}\e[0m"
      lines[1..-1].zip(colors) do |line, color|
        if color
          s << "\e[38;05;#{color}m"
        end

        s << line

        s << "\e[0m"
      end

      print s
      next
    end

    _, dst, bps_in, bps_out, total_in, total_out = line.split(";")

    if ARGV.include?("-n") || dst == "total"
      hostname = dst
    else
      hostname = hostnames[dst] ||= `~/bin/ip-to-hostname '#{dst}'`.chomp
    end

    bps_in    = bps_in.to_i
    bps_out   = bps_out.to_i
    total_in  = total_in.to_i
    total_out = total_out.to_i

    if bps_in == 0 && bps_out == 0
      colors << nil
    else
      if bps_in / MAX_IN.to_f > bps_out / MAX_OUT.to_f
        # IN is the culprit
        color_index = bps_in / (MAX_IN / COLORS.size)
      else
        # OUT is the culprit
        color_index = bps_out / (MAX_OUT / COLORS.size)
      end

      # TODO check if we are truncating our color spectrum

      if color_index >= COLORS.size
        color_index = COLORS.size - 1
      end

      colors << COLORS[color_index]
    end

    output << "%-20.20s\t%11s/s\t%11s/s\t%10s\t%10s" % [
                                                        hostname,
                                                        Filesize.new(bps_in).pretty,
                                                        Filesize.new(bps_out).pretty,
                                                        Filesize.new(total_in).pretty,
                                                        Filesize.new(total_out).pretty
                                                       ]
  end
end
