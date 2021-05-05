class Server
    def initialize(dir = '.', port = 6767)
        @pid = spawn("../fly-server -port #{port} #{dir}")
    end

    def kill
        Process.kill('TERM', @pid)
        Process.wait(@pid)
    end
end