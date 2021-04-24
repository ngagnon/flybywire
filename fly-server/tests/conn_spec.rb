require 'resp'
require 'server'
require 'fileutils'
require 'tmpdir'

RSpec.describe 'Connection' do
    before(:all) do
        @dir = Dir.mktmpdir 'fly'
        @s = Server.new @dir
    end

    after(:all) do
        @s.kill
        FileUtils.rm_rf @dir
    end

    before(:each) do
        @r = RESP.new
    end

    after(:each) do
        @r.close()
    end

    describe 'PING' do
        it 'returns PONG' do
            @r.put_array('PING')
            line = @r.get_simple_str
            expect(line).to eq('PONG')
        end

        it 'is case insensitive' do
            @r.put_array('pinG')
            line = @r.get_simple_str
            expect(line).to eq('PONG')
        end
    end

    describe 'QUIT' do
        it 'returns OK' do
            @r.put_array('QUIT')
            line = @r.get_simple_str
            expect(line).to eq('OK')
        end

        it 'cancels all pipelined commands' do
            @r.buffer do |b|
                b.put_array("MKDIR", "hello")
                b.put_array("QUIT")
                b.put_array("MKDIR", "world")
            end

            @r.get_simple_str
            @r.get_simple_str

            newdir = File.join(@dir, 'hello')
            expect(Dir.exist? newdir).to be true

            newdir = File.join(@dir, 'world')
            expect(Dir.exist? newdir).to be false
        end
    end
end