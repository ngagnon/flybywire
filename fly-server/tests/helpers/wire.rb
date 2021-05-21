module Wire
    class Integer
        def initialize(val)
            @val = val
        end

        def put(s)
            s.puts ":#{@val}\n"
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
            s = s.read(len)
            s.gets("\n")
            return [:blob, s]
        elsif line.start_with? '%'
            line.delete_prefix!("%")
            num_pairs = line.to_i
            map = {}

            num_pairs.times do
                (type, key) = get_next(s)

                if type != :blob && type != :string
                    raise 'map keys should be string or blob'
                end

                val = get_next(s)

                map[key] = val
            end

            return [:map, map]
        elsif line.start_with? '>'
            line.delete_prefix!(">")
            stream_id = line.to_i
            payload = get_next(s)

            return [:frame, Frame.new(stream_id, payload)]
        elsif line.start_with? '#'
            return [:bool, line[1] == 't']
        elsif line == '_'
            return [:null, nil]
        else
            raise 'get_next: illegal data type: ' + line[0]
        end
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
