require 'socket'
require 'benchmark'
require 'fileutils'
require 'tmpdir'

RSpec.configure do |config|
    config.before(:suite) do
        $dir = Dir.mktmpdir 'fly'
        @s = Server.new $dir
        @r = RESP.new

        @r.put_array('ADDUSER', 'example', 'supersecret')
        @r.get_next

        @r.close
        @s.kill

        $s = Server.new $dir

        $admin = RESP.new
        $admin.put_array('AUTH', 'PWD', 'example', 'supersecret')
        $admin.get_string

        $unauth = RESP.new
    end

    config.after(:suite) do
        $admin.close
        $unauth.close
        $s.kill
        FileUtils.rm_rf $dir
    end
end

class Server
    def initialize(dir = '.', port = 6767)
        @pid = spawn("../fly-server -port #{port} #{dir}")
    end

    def kill
        Process.kill('TERM', @pid)
        Process.wait(@pid)
    end
end

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

    def put_stream(id)
        @s.puts ">#{id}\n"
    end

    def put_null()
        @s.puts "_\n"
    end

    def pub_blob(blob)
        @s.puts "$#{blob.length}\n"
        @s.puts "#{blob}\n"
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
        elsif line.start_with? ':'
            line.delete_prefix!(':')
            return [:int, line.to_i]
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

require 'test-prof'

TestProf.configure do |config|
    # the directory to put artifacts (reports) in ('tmp/test_prof' by default)
    config.output_dir = "./test_prof"
  
    # use unique filenames for reports (by simply appending current timestamp)
    config.timestamps = true
  
    # color output
    config.color = true
end

TestProf::RubyProf.configure do |config|
    config.printer = :call_stack
end

#TestProf::RubyProf.run