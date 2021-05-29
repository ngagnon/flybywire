require 'socket'
require 'benchmark'
require 'fileutils'
require 'tmpdir'
require 'test-prof'

require_relative 'helpers/server'
require_relative 'helpers/session'
require_relative 'helpers/wire'
require_relative 'helpers/username'

# @TODO: regularUser not setup
module TestSuite
    def self.setup()
        $dir = Dir.mktmpdir 'fly'
        @@server = Server.new $dir

        session = Session.new
        session.cmd!('ADDUSER', 'example', 'supersecret')
        session.close

        @@admin = Session.new
        @@admin.cmd!('AUTH', 'PWD', 'example', 'supersecret')

        @@unauth = Session.new
    end

    def self.teardown()
        @@admin.close
        @@unauth.close
        @@server.kill
        FileUtils.rm_rf $dir
    end

    def admin()
        @@admin
    end

    def unauth()
        @@unauth
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