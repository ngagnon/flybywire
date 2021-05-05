require 'resp'
require 'server'
require 'fileutils'
require 'tmpdir'

RSpec.describe 'Connection' do
    describe 'PING' do
        context 'authenticated' do
            it 'returns PONG' do
                $admin.put_array('PING')
                line = $admin.get_string
                expect(line).to eq('PONG')
            end
        end

        context 'unauthenticated' do
            it 'returns PONG' do
                $unauth.put_array('PING')
                line = $unauth.get_string
                expect(line).to eq('PONG')
            end
        end

        it 'is case insensitive' do
            $unauth.put_array('pinG')
            line = $unauth.get_string
            expect(line).to eq('PONG')
        end
    end

    describe 'QUIT' do
        context 'authenticated' do
            before(:each) do
                @r = RESP.new
                @r.put_array('AUTH', 'PWD', 'example', 'supersecret')
                @r.get_next
            end

            after(:each) do
                @r.close
            end

            it 'returns OK' do
                @r.put_array('QUIT')
                line = @r.get_string
                expect(line).to eq('OK')
            end

            it 'cancels all pipelined commands' do
                @r.buffer do |b|
                    b.put_array("MKDIR", "hello")
                    b.put_array("QUIT")
                    b.put_array("MKDIR", "world")
                end

                @r.get_string
                @r.get_string

                newdir = File.join($dir, 'hello')
                expect(Dir.exist? newdir).to be true

                newdir = File.join($dir, 'world')
                expect(Dir.exist? newdir).to be false
            end
        end

        context 'unauthenticated' do
            before(:each) do
                @r = RESP.new
            end

            after(:each) do
                @r.close
            end

            it 'returns OK' do
                @r.put_array('QUIT')
                line = @r.get_string
                expect(line).to eq('OK')
            end
        end
    end
end