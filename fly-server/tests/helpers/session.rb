require_relative 'wire'

class SessionIO
    def put_array(*items)
        @s.puts "*#{items.length}\n"

        items.each do |s|
            if s.respond_to?(:put)
                s.put(@s)
            else
                @s.puts "$#{s.length}\n"
                @s.puts "#{s}\n"
            end
        end
    end

    def put_stream(id)
        @s.puts ">#{id}\n"
    end

    def put_null()
        @s.puts "_\n"
    end

    def put_blob(blob)
        @s.puts "$#{blob.length}\n"
        @s.puts "#{blob}\n"
    end
end

class SessionBuffer < SessionIO
    def initialize()
        @s = StringIO.new
    end

    def flush(sock)
        sock.puts @s.string
    end
end

class Session < SessionIO
    def initialize(port = 6767)
        5.times do
            begin
                @s = TCPSocket.new('localhost', port)
                break
            rescue
                sleep 0.100
            end
        end

        raise 'could not open connection' unless @s
    end

    def close
        @s.close()
        @s = nil
    end

    def buffer()
        buf = SessionBuffer.new
        yield(buf)
        buf.flush(@s)
    end

    def get_next()
        Wire.get_next(@s)
    end

    def cmd(name, *items)
        arr = Wire::Array.new([name].concat(items))
        arr.put(@s)

        get_next
    end

    def cmd!(name, *items)
        resp = cmd(name, *items)

        if resp.instance_of? Wire::Error
            raise "unexpected error: #{resp.code}: #{resp.msg}"
        end
    end

    def get_int()
        val = get_next()

        if !val.instance_of? Wire::Integer
            raise 'get_int: did not get an integer'
        end

        val.value
    end

    def get_string()
        val = get_next()

        if !val.instance_of? Wire::String
            raise "get_string: did not get a string, got #{val.class}"
        end

        val.value
    end

    def get_error()
        val = get_next()

        if !val.instance_of? Wire::Error
            raise 'get_error: did not get an error'
        end

        val.code + ' ' + val.msg
    end

    def get_blob()
        val = get_next()

        if !val.instance_of? Wire::Blob
            raise 'get_blob: did not get a blob'
        end

        val.value
    end

    def get_str_or_blob()
        val = get_next()

        if (!val.instance_of? Wire::Blob) && (!val.instance_of? Wire::String)
            raise 'get_str_or_blob: did not get a string or a blob'
        end

        val.value
    end

    def get_map()
        val = get_next()

        if !val.instance_of? Wire::Map
            raise "get_map: did not get a map, got '#{val}'"
        end

        val
    end
end
