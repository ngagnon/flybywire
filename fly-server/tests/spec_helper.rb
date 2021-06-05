require 'socket'
require 'benchmark'
require 'fileutils'
require 'tmpdir'
require 'test-prof'

require_relative 'helpers/server'
require_relative 'helpers/session'
require_relative 'helpers/wire'
require_relative 'helpers/username'

module TestSuite
    def self.setup()
        @@commands = []

        check_connection(6767)
        check_connection(6701)

        @@singleUserDir = Dir.mktmpdir 'fly'
        @@singleUserServer = Server.new(@@singleUserDir, 6701)
        @@singleUser = Session.new(6701)

        $dir = Dir.mktmpdir 'fly'
        @@server = Server.new $dir

        session = Session.new
        session.cmd!('ADDUSER', 'example', 'supersecret')
        session.close

        @@admin = Session.new
        @@admin.cmd!('AUTH', 'PWD', 'example', 'supersecret')
        @@admin.cmd!('ADDUSER', 'joe', 'regularguy')

        @@regularUser = Session.new
        @@regularUser.cmd!('AUTH', 'PWD', 'joe', 'regularguy')

        @@unauth = Session.new
    end

    def self.check_connection(port)
        connected = true

        begin
            @s = TCPSocket.new('localhost', port)
        rescue
            connected = false
        end

        if connected
            raise "A fly server is already running on port #{port}"
        end
    end

    def self.get_command(name)
        if @@commands.include? name
            name = name.downcase
        end

        @@commands.push(name)
        name
    end

    def self.teardown()
        @@admin.close
        @@regularUser.close
        @@unauth.close
        @@singleUser.close

        @@server.kill
        FileUtils.rm_rf $dir

        @@singleUserServer.kill
        FileUtils.rm_rf @@singleUserDir

        normalCase = @@commands.select { |x| x == x.upcase }
        normalCase.each do |cmd|
            hasAltCase = @@commands.any? { |x| x.upcase == cmd && x != cmd }

            if !hasAltCase
                raise "Command #{cmd} was only called in uppercase."
            end
        end
    end

    def admin()
        @@admin
    end

    def unauth()
        @@unauth
    end

    def regularUser()
        @@regularUser
    end

    def singleUser()
        @@singleUser
    end
end

RSpec.configure do |config|
    config.include TestSuite

    config.before(:suite) do
        TestSuite.setup
    end

    config.after(:suite) do
        TestSuite.teardown
    end
end

RSpec::Matchers.define :be_ok do
    match do |resp|
        (resp.is_a? Wire::String) && resp.value == 'OK'
    end
end

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