require 'fileutils'
require 'tmpdir'

RSpec.describe 'Authentication' do
    it 'disallows unauthenticated access' do
        resp = unauth.cmd('MKDIR', 'hello/world')
        expect(resp).to be_a(Wire::Error)
        expect(resp.code).to eq('DENIED')
    end

    describe 'AUTH' do
        context 'valid password' do
            before(:all) do
                @session = Session.new
                @resp = @session.cmd('AUTH', 'PWD', 'example', 'supersecret')
            end

            after(:all) do
                @session.close
            end

            it 'returns OK' do
                expect(@resp).to be_a(Wire::String)
                expect(@resp.value).to eq('OK')
            end

            it 'logs in the user' do
                resp = @session.cmd('WHOAMI')
                expect(resp).to be_a(Wire::String)
                expect(resp.value).to eq('example')
            end

            it 'lets user run commands' do
                @session.cmd!('MKDIR', 'hello/world')
            end
        end

        context 'invalid password' do
            before(:all) do
                @session = Session.new
                @resp = @session.cmd('AUTH', 'PWD', 'example', 'wrongpassword')
            end

            after(:all) do
                @session.close
            end

            it 'returns DENIED' do
                expect(@resp).to be_a(Wire::Error)
                expect(@resp.code).to eq('DENIED')
            end

            it 'does not log you in' do
                resp = @session.cmd('WHOAMI')
                expect(resp).to be_a(Wire::Null)
            end
        end
    end
end