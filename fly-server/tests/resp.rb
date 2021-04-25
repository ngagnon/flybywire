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
            return [:simple_string, line]
        elsif line.start_with? '-'
            line.delete_prefix!("-")
            return [:error_string, line]
        elsif line.start_with? '$'
            line.delete_prefix!("$")
            len = line.to_i
            s = @s.read(len)
            @s.gets("\n")
            return [:bulk_string, s]
        elsif line.start_with? '%'
            line.delete_prefix!("%")
            num_pairs = line.to_i
            map = {}

            num_pairs.times do
                key = get_str()
                val = get_next()

                map[key] = val
            end

            return [:map, map]
        elsif line == '1' || line == '0'
            return [:bool, line == '1']
        elsif line == '_'
            return [:null, nil]
        else
            raise 'get_next: illegal data type'
        end
    end

    def get_simple_str()
        (type, val) = get_next()

        if type != :simple_string
            raise 'get_simple_str: did not get a simple string'
        end

        val
    end

    def get_error_str()
        (type, val) = get_next()

        if type != :error_string
            raise 'get_error_str: did not get an error string'
        end

        val
    end

    def get_bulk_str()
        (type, val) = get_next()

        if type != :bulk_string
            raise 'get_bulk_str: did not get a bulk string'
        end

        val
    end

    def get_str()
        (type, val) = get_next()

        if type != :bulk_string && type != :simple_string
            raise 'get_str: did not get a string'
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