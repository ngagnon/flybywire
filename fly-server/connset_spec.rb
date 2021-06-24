require 'securerandom'

RSpec.describe 'CONNSET' do
    context 'authorized' do
        ['admin', 'regular user', 'single user'].each do |persona|
            context "as #{persona}" do
                before(:all) do
                    @session = as(persona)
                end

                [32768, 65536, 1048576].each do |size|
                    context "set ChunkSize = #{size}" do
                        before(:all) do
                            @resp = @session.cmd('CONNSET', 'ChunkSize', size)
                        end

                        it 'returns OK' do
                            expect(@resp).to be_ok
                        end

                        it 'updates the value' do
                            resp = @session.cmd!('CONNGET', 'ChunkSize')
                            expect(resp).to be_a(Wire::Integer)
                            expect(resp.value).to eq(size)
                        end

                        it 'updates the chunk size' do
                            # @TODO: try sending valid & invalid chunks
                        end
                    end
                end

                [0, -1].each do |size|
                    context "set ChunkSize = #{size}" do
                        it 'returns ARG' do
                            resp = @session.cmd('CONNSET', 'ChunkSize', size)
                            expect(resp).to be_error('ARG')
                        end
                    end
                end

                context "set ChunkSize with open write stream" do
                    it 'returns ILLEGAL' do
                        streamResp = @session.cmd!('STREAM', 'W', "connset-#{SecureRandom.hex}.txt")
                        resp = @session.cmd('CONNSET', 'ChunkSize', 1234)
                        expect(resp).to be_error('ILLEGAL')
                        @session.cmd!('CLOSE', streamResp.value)
                    end
                end

                context "set ChunkSize with open read stream" do
                    it 'returns ILLEGAL' do
                        streamResp = @session.cmd!('STREAM', 'R', 'big-file.txt')
                        resp = @session.cmd('CONNSET', 'ChunkSize', 1234)
                        expect(resp).to be_error('ILLEGAL')
                        @session.cmd!('CLOSE', streamResp.value)
                    end
                end

                context 'set invalid key' do
                    it 'returns ARG' do
                        resp = @session.cmd('CONNSET', 'Hello', 'Value')
                        expect(resp).to be_error('ARG')
                    end
                end
            end
        end
    end
    
    context 'unauthenticated' do
        it 'returns ILLEGAL' do
            resp = unauth.cmd('CONNSET', 'Hello', 'Value')
            expect(resp).to be_error('ILLEGAL')
        end
    end
end