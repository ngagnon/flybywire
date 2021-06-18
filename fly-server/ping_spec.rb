RSpec.describe 'PING' do
    context 'authenticated' do
        it 'returns PONG' do
            resp = admin.cmd('PING')
            expect(resp).to be_a(Wire::String)
            expect(resp.value).to eq('PONG')
        end
    end

    context 'unauthenticated' do
        it 'returns PONG' do
            resp = unauth.cmd('PING')
            expect(resp).to be_a(Wire::String)
            expect(resp.value).to eq('PONG')
        end
    end
end