require 'fileutils'
require 'tmpdir'

RSpec.describe 'Single-user mode' do
    before(:all) do
        @dir = Dir.mktmpdir 'fly'
        @s = Server.new(@dir, 6868)
        @r = Session.new(6868)
    end

    after(:all) do
        @r.close
        @s.kill
        FileUtils.rm_rf @dir
    end

    it 'allows all operations' do
        @r.put_array('PING')
        line = @r.get_string
        expect(line).to eq('PONG')

        @r.put_array('MKDIR', 'hello/world')
        line = @r.get_string
        expect(line).to eq('OK')
    end

    context 'first user created' do
        before(:all) do
            @r.put_array('ADDUSER', 'example', 'supersecret')
            line = @r.get_string
            expect(line).to eq('OK')
        end

        it 'is created as admin' do
            @r.put_array('SHOWUSER', 'example')
            data = @r.get_map
            expect(data['username']).to be_a(Wire::String)
            expect(data['username'].value).to eq('example')
            expect(data['admin']).to be_a(Wire::Boolean)
            expect(data['admin'].value).to be(true)
        end

        it 'becomes current user' do
            @r.put_array('WHOAMI')
            line = @r.get_string
            expect(line).to eq('example')
        end

        it 'disallows unauthenticated operations' do
            @r2 = Session.new

            @r2.put_array('MKDIR', 'hello/world')
            line = @r2.get_error
            expect(line).to start_with('DENIED')

            @r2.close
        end
    end
end