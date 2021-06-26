require 'securerandom'

RSpec.describe 'COPY' do
    context 'unauthorized' do
        it 'returns DENIED' do
            resp = unauth.cmd('COPY', 'copy-src.txt', 'copy-dst.txt')
            expect(resp).to be_error('DENIED')
        end
    end

    context 'authorized' do
        ['admin', 'regular user', 'single user'].each do |persona|
            context "as #{persona}" do
                before(:all) do
                    @session = as(persona)
                end

                context 'file' do
                    before(:all) do
                        @src = "copy-from-#{SecureRandom.hex}.txt"
                        @dst = "copy-to-#{SecureRandom.hex}.txt"
                        @data = "hello\nworld\ncopy\n" * 31 * 1024
                        @session.write_file(@src, @data)
                        @resp = @session.cmd('COPY', @src, @dst)
                    end

                    it 'returns stream ID' do
                        expect(@resp).to be_a(Wire::Integer)
                    end

                    it 'copies file' do
                        resp = @session.get_next
                        expect(resp).to be_a(Wire::Frame)
                        expect(resp.id).to eq(@resp.value)
                        expect(resp.payload).to be_a(Wire::Null)

                        resp = @session.cmd('LIST', @src)
                        expect(resp).to be_a(Wire::Table)
                        expect(resp.row_count).to eq(1)

                        resp = @session.cmd('LIST', @dst)
                        expect(resp).to be_a(Wire::Table)
                        expect(resp.row_count).to eq(1)

                        contents = @session.read_file(@dst)
                        expect(contents == @data).to be(true)
                    end
                end

                context 'folder' do
                    it 'returns ARG' do
                        src = "copy-from-#{SecureRandom.hex}"
                        @session.cmd!('MKDIR', src)
                        @session.write_file("#{src}/hello.txt", "hello\mworld\ncopy\nfolder")
                        dst = "copy-to-#{SecureRandom.hex}"

                        resp = @session.cmd('COPY', src, dst)
                        expect(resp).to be_error('ARG')
                    end
                end
            end
        end
    end
end