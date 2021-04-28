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
            line = @r.get_string
            expect(line).to eq('OK')
        end

        context 'first user created' do
            before(:each) do
                @r.put_array('ADDUSER', 'example', 'supersecret')
                line = @r.get_string
                expect(line).to eq('OK')
            end

            it 'is created as admin' do
                @r.put_array('SHOWUSER', 'example')
                data = @r.get_map
                expect(data['username'][0]).to eq(:string)
                expect(data['username'][1]).to eq('example')
                expect(data['admin'][0]).to eq(:bool)
                expect(data['admin'][1]).to eq(true)
            end

            it 'becomes current user' do
                @r.put_array('WHOAMI')
                line = @r.get_string
                expect(line).to eq('example')
            end

            it 'becomes impossible to connect unauthenticated' do
                @r2 = RESP.new

                @r2.put_array('MKDIR', 'hello/world')
                line = @r2.get_error
                expect(line).to start_with('DENIED')

                @r2.close
            end
        end
    end

    context 'with fly database' do
        before(:all) do
            @dir = Dir.mktmpdir 'fly'
            @s = Server.new @dir
            @r = RESP.new

            @r.put_array('ADDUSER', 'example', 'supersecret')
            @r.get_next

            @r.put_array('QUIT')
            @r.get_next

            @r.close
            @s.kill

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
            @r.close
        end

        it 'disallows unauthenticated access' do
            @r.put_array('MKDIR', 'hello/world')
            line = @r.get_error
            expect(line).to start_with('DENIED')
        end

        it 'allows unauthenticated ping' do
            @r.put_array('PING')
            line = @r.get_string
            expect(line).to eq('PONG')
        end

        it 'allows unauthenticated quit' do
            @r.put_array('QUIT')
            line = @r.get_string
            expect(line).to eq('OK')
        end

        describe 'AUTH' do
            it 'returns OK' do
                @r.put_array('AUTH', 'PWD', 'example', 'supersecret')
                line = @r.get_string
                expect(line).to eq('OK')
            end

            it 'logs in the user' do
                @r.put_array('AUTH', 'PWD', 'example', 'supersecret')
                line = @r.get_string
                expect(line).to eq('OK')

                @r.put_array('WHOAMI')
                (type, val) = @r.get_next
                expect(type).to eq(:string)
                expect(val).to eq('example')
            end

            it 'verifies the supplied password' do
                @r.put_array('AUTH', 'PWD', 'example', 'wrongpassword')
                line = @r.get_error
                expect(line).to start_with('DENIED')

                @r.put_array('WHOAMI')
                (type, val) = @r.get_next
                expect(type).to eq(:null)
            end
        end

        context 'user logged in' do
            before(:each) do
                @r.put_array('AUTH', 'PWD', 'example', 'supersecret')
                @r.get_next
            end

            it 'is allowed to run commands' do
                @r.put_array('MKDIR', 'hello/world')
                line = @r.get_string
                expect(line).to eq('OK')
            end
        end
    end
end