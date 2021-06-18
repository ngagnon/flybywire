RSpec.describe 'MKDIR' do
    context 'authorized' do
        before(:all) do
            @resp = admin.cmd('MKDIR', 'world')
        end

        it 'returns OK' do
            expect(@resp).to be_a(Wire::String)
            expect(@resp.value).to eq('OK')
        end

        it 'creates a folder' do
            newdir = File.join($dir, 'world')
            expect(Dir.exist? newdir).to be true
        end
    end

    context 'unauthorized' do
        # @TODO: OK: single-user
        # @TODO: OK: user is admin
        # @TODO: OK: with valid ACP
        # @TODO: DENIED: unauthenticated user
        # @TODO: DENIED: user doesn't exist anymore
        # @TODO: DENIED: without a valid ACP
    end
end