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

    def get_simple_str()
        line = @s.gets("\n")
        line.delete_prefix!("+")
        line.delete_suffix!("\n")
        line
    end

    def get_error_str()
        line = @s.gets("\n")
        line.delete_prefix!("-")
        line.delete_suffix!("\n")
        line
    end

    def get_bulk_str()
        line = gets()
        line.delete_prefix!("$")

        len = line.to_i
        data = @s.read(len)
        gets()

        data
    end
end