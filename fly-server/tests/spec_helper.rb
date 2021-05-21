require 'socket'
require 'benchmark'
require 'fileutils'
require 'tmpdir'
require 'test-prof'

require_relative 'helpers/server'
require_relative 'helpers/session'
require_relative 'helpers/wire'

RSpec.configure do |config|
    config.before(:suite) do
        $dir = Dir.mktmpdir 'fly'
        @s = Server.new $dir
        @r = Session.new

        @r.put_array('ADDUSER', 'example', 'supersecret')
        @r.get_next

        @r.close
        @s.kill

        $s = Server.new $dir

        $admin = Session.new
        $admin.put_array('AUTH', 'PWD', 'example', 'supersecret')
        $admin.get_string

        $unauth = Session.new
    end

    config.after(:suite) do
        $admin.close
        $unauth.close
        $s.kill
        FileUtils.rm_rf $dir
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