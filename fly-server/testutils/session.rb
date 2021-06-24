require_relative 'wire'

class SessionIO
    def put_array(*items)
        @s.puts "*#{items.length}\n"

        items.each do |s|
            if s.respond_to?(:put)
                s.put(@s)
            else
                @s.puts "+#{s}\n"
            end
        end
    end

    def put_stream(id)
        @s.puts "@#{id}\n"
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
    attr_reader :label

    def initialize(**opts)
        port = opts[:port] || 6767
        @label = opts[:label] || 'no name'

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
        name = TestSuite.get_command(name)
        arr = Wire::Array.new([name].concat(items))
        arr.put(@s)

        loop do
            v = get_next

            if !(v.is_a? Wire::Frame)
                return v
            end
        end
    end

    def cmd!(name, *items)
        resp = cmd(name, *items)

        if resp.instance_of? Wire::Error
            raise "unexpected error: #{resp.code}: #{resp.msg}"
        end

        resp
    end

    def read_file(name)
        resp = cmd!('STREAM', 'R', name)
        id = resp.value
        contents = ''

        while true
            resp = get_next()

            if !(resp.is_a? Wire::Frame)
                raise 'response was expected to be a stream frame'
            end

            if resp.id != id
                raise "unexpected frame id #{id}"
            end

            if resp.payload.is_a? Wire::Null
                return contents
            end

            if !(resp.payload.is_a? Wire::Blob)
                raise 'expected stream frame to be null or blob'
            end

            contents << resp.payload.value
        end
    end

    def write_file(name, contents)
        resp = cmd!('STREAM', 'W', name)
        id = resp.value

        put_stream(id)
        put_blob(contents)

        put_stream(id)
        put_null

        50.times do
            resp = cmd('LIST', name)

            if resp.is_a? Wire::Error
                sleep 0.020
            else
                break
            end
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
