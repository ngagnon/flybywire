require_relative 'resp'
require_relative 'server'
require 'fileutils'
require 'tmpdir'

RSpec.describe 'Authentication' do
    context 'no fly database' do
        before(:each) do
            @dir = Dir.mktmpdir 'fly'
            @s = Server.new @dir
            @r = RESP.new
        end

        after(:each) do
            @r.close
            @s.kill
            FileUtils.rm_rf @dir
        end

        it 'allows all operations (single-user mode)' do
            @r.put_array('MKDIR', 'hello/world')
            line = @r.get_simple_str
            expect(line).to eq('OK')
        end

        context 'first user created' do
            before(:each) do
                @r.put_array('ADDUSER', 'example', 'supersecret')
                line = @r.get_simple_str
                expect(line).to eq('OK')
            end

            xit 'is created as admin' do
                @r.put_array('SHOWUSER', 'example')
                data = @r.get_map
                expect(data[:username]).to eq('example')
                expect(data[:admin]).to eq(true)
            end

            it 'becomes current user' do
                @r.put_array('WHOAMI')
                line = @r.get_simple_str
                expect(line).to eq('example')
            end

            it 'becomes impossible to connect unauthenticated' do
                @r2 = RESP.new

                @r2.put_array('MKDIR', 'hello/world')
                line = @r2.get_error_str
                expect(line).to eq('DENIED')

                @r2.close
            end
        end
    end

    context 'with fly database' do
        before(:each) do
            @dir = Dir.mktmpdir 'fly'
            @s = Server.new @dir
            @r = RESP.new

            # @TODO: create user, then kill server and restart it
        end

        after(:each) do
            @r.close
            @s.kill
            FileUtils.rm_rf @dir
        end

        xit 'disallows unauthenticated access' do
            @r.put_array('MKDIR', 'hello/world')
            line = @r.get_error_str
            expect(line).to eq('DENIED')
        end

        xit 'allows unauthenticated ping' do
            @r.put_array('PING')
            line = @r.get_simple_str
            expect(line).to eq('PONG')
        end

        xit 'allows unauthenticated quit' do
            @r.put_array('QUIT')
            line = @r.get_simple_str
            expect(line).to eq('OK')
        end

        context 'user logs in' do
            # @TODO: auth command!

            xit 'is allowed to run commands' do
                @r.put_array('MKDIR', 'hello/world')
                line = @r.get_simple_str
                expect(line).to eq('OK')
            end
        end
    end
end