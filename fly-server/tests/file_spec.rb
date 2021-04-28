require_relative 'resp'
require_relative 'server'
require 'fileutils'
require 'tmpdir'

RSpec.describe 'File commands' do
    before(:all) do
        @dir = Dir.mktmpdir 'fly'
        @s = Server.new @dir
        @r = RESP.new
    end

    after(:all) do
        @r.close
        @s.kill
        FileUtils.rm_rf @dir
    end

    describe 'MKDIR' do
        before(:all) do
            @r.put_array('MKDIR', 'world')
            @line = @r.get_string
        end

        it 'returns OK' do
            expect(@line).to eq('OK')
        end

        it 'creates a folder' do
            newdir = File.join(@dir, 'world')
            expect(Dir.exist? newdir).to be true
        end
    end
end