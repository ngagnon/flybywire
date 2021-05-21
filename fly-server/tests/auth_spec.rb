require 'fileutils'
require 'tmpdir'

RSpec.describe 'Authentication' do
    it 'disallows unauthenticated access' do
        $unauth.put_array('MKDIR', 'hello/world')
        line = $unauth.get_error
        expect(line).to start_with('DENIED')
    end

    describe 'AUTH' do
        context 'valid password' do
            before(:all) do
                @r = Session.new
                @r.put_array('AUTH', 'PWD', 'example', 'supersecret')
                @line = @r.get_string
            end

            after(:all) do
                @r.close
            end

            it 'returns OK' do
                expect(@line).to eq('OK')
            end

            it 'logs in the user' do
                @r.put_array('WHOAMI')
                (type, val) = @r.get_next
                expect(type).to eq(:string)
                expect(val).to eq('example')
            end

            it 'lets user run commands' do
                @r.put_array('MKDIR', 'hello/world')
                line = @r.get_string
                expect(line).to eq('OK')
            end
        end

        context 'invalid password' do
            before(:all) do
                @r = Session.new
                @r.put_array('AUTH', 'PWD', 'example', 'wrongpassword')
                @line = @r.get_error
            end

            after(:all) do
                @r.close
            end

            it 'returns DENIED' do
                expect(@line).to start_with('DENIED')
            end

            it 'does not log you in' do
                @r.put_array('WHOAMI')
                (type, val) = @r.get_next
                expect(type).to eq(:null)
            end
        end
    end
end