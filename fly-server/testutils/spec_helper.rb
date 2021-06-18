require 'socket'
require 'benchmark'
require 'fileutils'
require 'tmpdir'
require 'test-prof'

require_relative 'server'
require_relative 'session'
require_relative 'wire'
require_relative 'username'

module TestSuite
    def self.setup()
        @@commands = []

        check_connection(6767)
        check_connection(6701)

        @@single_user_dir = Dir.mktmpdir 'fly'
        @@single_user_server = Server.new(@@single_user_dir, 6701)
        @@single_user = Session.new(port: 6701, label: 'single user')

        $dir = Dir.mktmpdir 'fly'
        @@server = Server.new $dir

        session = Session.new
        session.cmd!('ADDUSER', 'example', 'supersecret')
        session.close

        @@admin = Session.new(label: 'admin')
        @@admin.cmd!('AUTH', 'PWD', 'example', 'supersecret')
        @@admin.cmd!('ADDUSER', 'joe', 'regularguy')

        @@regular_user = Session.new(label: 'regular user')
        @@regular_user.cmd!('AUTH', 'PWD', 'joe', 'regularguy')

        @@unauth = Session.new(label: 'unauthenticated')
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
        @@regular_user.close
        @@unauth.close
        @@single_user.close

        @@server.kill
        FileUtils.rm_rf $dir

        @@single_user_server.kill
        FileUtils.rm_rf @@single_user_dir

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

    def regular_user()
        @@regular_user
    end

    def single_user()
        @@single_user
    end

    def as(key)
        if key == 'single user'
            single_user
        elsif key == 'regular user'
            regular_user
        elsif key == 'admin'
            admin
        elsif key == 'unauthenticated'
            unauth
        else
            raise "no such session: #{key}"
        end
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

RSpec::Matchers.define :be_error do |code|
    match do |resp|
        (resp.is_a? Wire::Error) && resp.code == code
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