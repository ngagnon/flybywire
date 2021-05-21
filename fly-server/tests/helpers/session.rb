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

    def get_int()
        (type, val) = get_next()

        if type != :int
            raise 'get_int: did not get an integer'
        end

        val
    end

    def get_string()
        (type, val) = get_next()

        if type != :string
            raise "get_string: did not get a string, got #{type.to_s} (#{val.to_s})"
        end

        val
    end

    def get_error()
        (type, val) = get_next()

        if type != :error
            raise 'get_error: did not get an error'
        end

        val
    end

    def get_blob()
        (type, val) = get_next()

        if type != :blob
            raise 'get_blob: did not get a blob'
        end

        val
    end

    def get_str_or_blob()
        (type, val) = get_next()

        if type != :blob && type != :string
            raise 'get_str_or_blob: did not get a string or a blob'
        end

        val
    end

    def get_map()
        (type, val) = get_next()

        if type != :map
            raise "get_map: did not get a map, got '#{val}'"
        end

        val
    end
end
