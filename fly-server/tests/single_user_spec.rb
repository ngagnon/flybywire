require 'fileutils'
require 'tmpdir'

RSpec.describe 'Single-user mode' do
    before(:all) do
        @dir = Dir.mktmpdir 'fly'
        @server = Server.new(@dir, 6868)
        @session = Session.new(6868)
    end

    after(:all) do
        @session.close
        @server.kill
        FileUtils.rm_rf @dir
    end

    it 'allows all operations' do
        @session.cmd!('PING')
        @session.cmd!('MKDIR', 'hello/world')
    end

    context 'first user created' do
        before(:all) do
            @session.cmd!('ADDUSER', 'example', 'supersecret')
        end

        it 'is created as admin' do
            resp = @session.cmd('SHOWUSER', 'example')
            expect(resp).to be_a(Wire::Map)
            expect(resp['username']).to be_a(Wire::String)
            expect(resp['username'].value).to eq('example')
            expect(resp['admin']).to be_a(Wire::Boolean)
            expect(resp['admin'].value).to be(true)
        end

        it 'becomes current user' do
            resp = @session.cmd('WHOAMI')
            expect(resp).to be_a(Wire::String)
            expect(resp.value).to eq('example')
        end

        it 'disallows unauthenticated operations' do
            @session2 = Session.new

            resp = @session2.cmd('MKDIR', 'hello/world')
            expect(resp).to be_a(Wire::Error)
            expect(resp.code).to eq('DENIED')

            @session2.close
        end
    end
end