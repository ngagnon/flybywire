class Server
    def initialize(dir = '.', port = 6767, opts = '-notls')
        if ENV['FLY_SHOW_OUTPUT']
            @pid = spawn("./fly-server -debug #{opts} -port #{port} #{dir}")
        else
            @pid = spawn("./fly-server -debug #{opts} -port #{port} #{dir}", :err => "/dev/null")
        end
    end

    def kill
        Process.kill('TERM', @pid)
        Process.wait(@pid)
    end
end

