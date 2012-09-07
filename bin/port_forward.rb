class PortForward < Struct.new(:ports, :protocol, :from, :to, :ttl)
  # @return [Array<String>] All iptables commands that are required to add/delete this forward
  def to_iptables(action)
    arguments_dnat = []
    arguments_forward = []

    case action
    when :add
      arguments_dnat << '-A PORTFORWARDS_DNAT'
      arguments_forward << '-A PORTFORWARDS_FORWARD'
    when :del, :delete
      arguments_dnat << '-D PORTFORWARDS_DNAT'
      arguments_forward << '-D PORTFORWARDS_FORWARD'
    else
      raise ArgumentError, "Unknown action. Allowed are :add and :del/:delete"
    end

    arguments_dnat << '-t nat'
    arguments_dnat << "-p '#{self.protocol}'"
    arguments_dnat << "--dport '#{self.ports.join(':')}'"
    arguments_dnat << '-i eth0'
    arguments_dnat << '-j DNAT'
    arguments_dnat << "--to-destination '#{self.to}'"

    arguments_forward << "-d '#{self.to.split(':').first}'"
    arguments_forward << "-p '#{self.protocol}'"
    # The FORWARD chain will be entered after DNAT, so the destinaion port has already been rewritten
    if self.to.include?(":")
      fdport = self.to.split(":").last.split("-").join(":")
    else
      fdport = self.ports.join(":")
    end
    arguments_forward << "--dport '#{fdport}'"
    arguments_forward << '-i eth0'
    arguments_forward << '-o eth1'
    arguments_forward << '-j ACCEPT'

    if self.ttl
      ttl_argument = "-m comment --comment 'sr_ttl=#{self.ttl}'"
      arguments_dnat << ttl_argument
      arguments_forward << ttl_argument
    end

    # FIXME Yes, this is definitely not safe against shell injection
    return ["sudo /sbin/iptables #{arguments_dnat.join(' ')}",
            "sudo /sbin/iptables #{arguments_forward.join(' ')}"]

  end

  # FIXME finish writing this method
  def to_remove_command
    if self.from != '0.0.0.0/0'
      remove = "remove-port-forward --ports '%s' -p '%s' -f '%s' -t '%s'" % [ports, protocol, source, to]
    else
      remove = "remove-port-forward --ports '%s' -p '%s' -t '%s'" % [ports, protocol, to]
      source = 'any'
    end

    ''
  end
end
