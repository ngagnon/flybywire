class Server
    def initialize(dir = '.')
        @pid = spawn("../fly-server #{dir}")
        puts "server spawned"
    end

    def kill
        Process.kill('TERM', @pid)
        Process.wait(@pid)
    end
end