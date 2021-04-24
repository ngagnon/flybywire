require 'resp'
require 'server'
require 'fileutils'
require 'tmpdir'

RSpec.describe 'File commands' do
    before(:each) do
        @dir = Dir.mktmpdir 'fly'
        @s = Server.new @dir
        @r = RESP.new
    end

    after(:each) do
        @r.close
        @s.kill
        FileUtils.remove_entry @dir
    end

    describe 'MKDIR' do
        it 'creates a folder' do
            @r.put_array('MKDIR', 'world')
            line = @r.get_simple_str
            expect(line).to eq('OK')

            newdir = File.join(@dir, 'world')
            expect(Dir.exist? newdir).to be true
        end
    end
end