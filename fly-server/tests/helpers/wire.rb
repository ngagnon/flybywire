module Wire
    class Integer
        attr_reader :value

        def initialize(value)
            @value = value
        end

        def put(s)
            s.puts ":#{@value}\n"
        end
    end

    class Boolean
        attr_reader :value

        def initialize(value)
            @value = value
        end

        def put(s)
            if @value
                s.puts "#t\n"
            else
                s.puts "#f\n"
            end
        end
    end

    class Array
        attr_reader :elems

        def initialize(elems)
            @elems = elems
        end

        def put(s)
            s.puts "*#{@elems.length}\n"

            @elems.each do |elem|
                if elem.is_a? ::String
                    Blob.new(elem).put(s)
                else
                    elem.put(s)
                end
            end
        end
    end

    class Map
        def initialize(value)
            @value = value
        end

        def [](key)
            @value[key]
        end
        
        def []=(key, value)
            @value[key] = value
        end

        def key?(key)
            @value.key?(key)
        end

        def keys()
            @value.keys
        end

        def put(s)
            s.puts "%#{@value.length}\n"

            @value.each do |key, value|
                key.put(s)
                value.put(s)
            end
        end
    end

    class String
        attr_reader :value

        def initialize(value)
            @value = value
        end

        def put(s)
            s.puts "+#{@value}\n"
        end
    end

    class Error
        attr_reader :code
        attr_reader :msg

        def initialize(code, msg)
            @code = code
            @msg = msg
        end

        def put(s)
            s.puts "-#{@code} #{@msg}\n"
        end
    end

    class Blob
        attr_reader :value

        def initialize(value)
            @value = value
        end

        def put(s)
            s.puts "$#{@value.length}\n"
            s.puts "#{@value}\n"
        end
    end

    class Null
        def put(s)
            s.puts "_\n"
        end
    end

    class Frame
        attr_reader :id
        attr_reader :payload

        def initialize(id, payload)
            @id = id
            @payload = payload
        end
    end

    def self.get_next(s)
        line = s.gets("\n")

        if line == nil
            raise 'unexpected end of file'
        end

        line.delete_suffix!("\n")

        if line.start_with? '+'
            line.delete_prefix!("+")
            return String.new(line)
        elsif line.start_with? '-'
            line.delete_prefix!("-")
            code = line[0, line.index(' ')]
            msg = line[line.index(' ') + 1..]
            return Error.new(code, msg)
        elsif line.start_with? ':'
            line.delete_prefix!(':')
            return Integer.new(line.to_i)
        elsif line.start_with? '$'
            line.delete_prefix!("$")
            len = line.to_i
            str = s.read(len)
            s.gets("\n")
            return Blob.new(str)
        elsif line.start_with? '*'
            line.delete_prefix!("*")
            num_elems = line.to_i
            elems = []

            num_elems.times do
                elem = get_next(s)
                elems.push(elem)
            end

            return Array.new(elems)
        elsif line.start_with? '%'
            line.delete_prefix!("%")
            num_pairs = line.to_i
            map = {}

            num_pairs.times do
                key = get_next(s)

                if (!key.instance_of? Blob) && (!key.instance_of? Wire::String)
                    raise 'map keys should be string or blob'
                end

                val = get_next(s)

                map[key.value] = val
            end

            return Map.new(map)
        elsif line.start_with? '>'
            line.delete_prefix!(">")
            stream_id = line.to_i
            payload = get_next(s)

            return Frame.new(stream_id, payload)
        elsif line.start_with? '#'
            return Boolean.new(line[1] == 't')
        elsif line == '_'
            return Null.new()
        else
            raise 'get_next: illegal data type: ' + line[0]
        end
    end
end