require 'socket'

class RESPIO
    def puts(s)
        if !s.end_with? "\n"
            s = s + "\n"
        end

        @s.puts s
    end

    def put_array(*items)
        @s.puts "*#{items.length}\n"

        items.each do |s|
            @s.puts "$#{s.length}\n"
            @s.puts "#{s}\n"
        end
    end
end

class BufferedRESP < RESPIO 
    def initialize()
        @s = StringIO.new
    end

    def flush(sock)
        sock.puts @s.string
    end
end

class RESP < RESPIO
    def initialize
        5.times do
            begin
                @s = TCPSocket.new 'localhost', 6767
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
        buf = BufferedRESP.new
        yield(buf)
        buf.flush(@s)
    end

    def get_next()
        line = @s.gets("\n")
        line.delete_suffix!("\n")

        if line.start_with? '+'
            line.delete_prefix!("+")
            return [:string, line]
        elsif line.start_with? '-'
            line.delete_prefix!("-")
            return [:error, line]
        elsif line.start_with? '$'
            line.delete_prefix!("$")
            len = line.to_i
            s = @s.read(len)
            @s.gets("\n")
            return [:blob, s]
        elsif line.start_with? '%'
            line.delete_prefix!("%")
            num_pairs = line.to_i
            map = {}

            num_pairs.times do
                key = get_str_or_blob()
                val = get_next()

                map[key] = val
            end

            return [:map, map]
        elsif line.start_with? '#'
            return [:bool, line[1] == 't']
        elsif line == '_'
            return [:null, nil]
        else
            raise 'get_next: illegal data type: ' + line[0]
        end
    end

    def get_string()
        (type, val) = get_next()

        if type != :string
            raise 'get_string: did not get a string'
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