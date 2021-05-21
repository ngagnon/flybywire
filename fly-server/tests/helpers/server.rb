class Server
    def initialize(dir = '.', port = 6767)
        @pid = spawn("../fly-server -debug -port #{port} #{dir}", :err => "/dev/null")
    end

    def kill
        Process.kill('TERM', @pid)
        Process.wait(@pid)
    end
end

