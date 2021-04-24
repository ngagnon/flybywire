require_relative 'resp'
require_relative 'server'
require_relative 'db'
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

            it 'is created as admin' do
                # @TODO
            end

            it 'becomes current user' do
                # @TODO
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
            FlyDB.setup @dir

            @s = Server.new @dir
            @r = RESP.new
        end

        after(:each) do
            @r.close
            @s.kill
            FileUtils.rm_rf @dir
        end

        it 'disallows unauthenticated access (multi-user mode)' do
            @r.put_array('MKDIR', 'hello/world')
            line = @r.get_error_str
            expect(line).to eq('DENIED')
        end

        it 'allows unauthenticated ping' do
            @r.put_array('PING')
            line = @r.get_simple_str
            expect(line).to eq('PONG')
        end

        it 'allows unauthenticated quit' do
            @r.put_array('QUIT')
            line = @r.get_simple_str
            expect(line).to eq('OK')
        end

        context 'user logs in' do
            # @TODO: auth command!

            it 'is allowed to run commands' do
                @r.put_array('MKDIR', 'hello/world')
                line = @r.get_simple_str
                expect(line).to eq('OK')
            end
        end
    end
end