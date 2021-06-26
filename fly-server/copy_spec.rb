require 'securerandom'

RSpec.describe 'COPY' do
    context 'file' do
        context 'authorized' do
            ['admin', 'regular user', 'single user'].each do |persona|
                context "as #{persona}" do
                    before(:all) do
                        @session = as(persona)
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
            end
        end

        context 'unauthorized' do
            it 'returns DENIED' do
                resp = unauth.cmd('COPY', 'copy-src.txt', 'copy-dst.txt')
                expect(resp).to be_error('DENIED')
            end
        end
    end

=begin
    context 'folder' do
        ['admin', 'regular user', 'single user'].each do |persona|
            context "as #{persona}" do
                before(:all) do
                    @session = as(persona)

                    @folder_name = "del-#{SecureRandom.hex}"
                    @session.cmd!('MKDIR', @folder_name)

                    @file_name = @folder_name + "/file.txt"
                    @session.write_file(@file_name, "hello\nworld\n")

                    @resp = @session.cmd('DEL', @folder_name)
                end

                it 'returns OK' do
                    expect(@resp).to be_ok
                end

                it 'deletes folder' do
                    resp = @session.cmd('LIST', "/" + @folder_name)
                    expect(resp).to be_error('NOTFOUND')
                end
            end
        end
    end
=end
end