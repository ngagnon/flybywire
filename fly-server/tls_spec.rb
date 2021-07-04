RSpec.describe 'TLS' do
    before(:all) do
        dir = Dir.mktmpdir 'fly'
        @server = Server.new(dir, 6768, '')
    end

    after(:all) do
        @server.kill
    end

    it 'accepts TLS connections' do
        fingerprint = Session.get_fingerprint(6768)

        s = Session.new(port: 6768, tls: true, fingerprint: fingerprint)

        resp = s.cmd('PING')
        expect(resp).to be_a(Wire::String)
        expect(resp.value).to eq('PONG')

        s.close
    end

    it 'does not accept invalid fingerprints' do
        error = false

        begin
            s = Session.new(port: 6768, tls: true, fingerprint: 'blahblah')
            s.close
        rescue
            error = true
        end

        expect(error).to be(true)
    end
end