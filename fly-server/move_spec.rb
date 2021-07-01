require 'securerandom'

RSpec.describe 'MOVE' do
    ['admin', 'regular user', 'single user'].each do |persona|
        context "as #{persona}" do
            context 'source exists' do
                before(:all) do
                    @session = as(persona)

                    @dir_name = "move-dst-#{SecureRandom.hex}"
                    @session.cmd!('MKDIR', @dir_name)
                    
                    @src = "move-src-#{SecureRandom.hex}.txt"
                    @session.write_file(@src, "hello\nworld\nmove\n")

                    @dst = "#{@dir_name}/move.txt"
                    @resp = @session.cmd('MOVE', @src, @dst)
                end

                it 'returns OK' do
                    expect(@resp).to be_ok
                end

                it 'moved file' do
                    resp = @session.cmd('LIST', @src)
                    expect(resp).to be_error('NOTFOUND')

                    resp = @session.cmd('LIST', @dst)
                    expect(resp).to be_a(Wire::Table)
                    expect(resp.row_count).to eq(1)

                    contents = @session.read_file(@dst)
                    expect(contents).to eq("hello\nworld\nmove\n")
                end
            end

            context 'source does not exist' do
                it 'returns NOTFOUND' do
                    @session = as(persona)
                    @src = "move-src-#{SecureRandom.hex}.txt"
                    @dir_name = "move-dst-#{SecureRandom.hex}"
                    @session.cmd!('MKDIR', @dir_name)
                    resp = @session.cmd('MOVE', @src, "#{@dir_name}/move.txt")
                    expect(resp).to be_error('NOTFOUND')
                end
            end

            context 'destination already exists' do
                before(:all) do
                    @session = as(persona)

                    @dir_name = "move-dst-#{SecureRandom.hex}"
                    @session.cmd!('MKDIR', @dir_name)
                    
                    @src = "move-src-#{SecureRandom.hex}.txt"
                    @session.write_file(@src, "hello\nworld\nmove\n")
                    
                    @dst = "#{@dir_name}/move-src.txt"
                    @session.write_file(@dst, "hello\nworld\nexists\n")

                    @resp = @session.cmd('MOVE', @src, @dst)
                end

                it 'returns OK' do
                    expect(@resp).to be_ok
                end

                it 'overrides destination' do
                    resp = @session.cmd('LIST', @src)
                    expect(resp).to be_error('NOTFOUND')

                    resp = @session.cmd('LIST', @dst)
                    expect(resp).to be_a(Wire::Table)
                    expect(resp.row_count).to eq(1)

                    contents = @session.read_file(@dst)
                    expect(contents).to eq("hello\nworld\nmove\n")
                end
            end
        end
    end

    context 'unauthorized' do
        context 'as unauthenticated' do
            it 'returns DENIED' do
                @session = unauth

                @dir_name = "move-dst-#{SecureRandom.hex}"
                admin.cmd!('MKDIR', @dir_name)
                
                @src = "move-src-#{SecureRandom.hex}.txt"
                admin.write_file(@src, "hello\nworld\nmove\n")

                @dst = "#{@dir_name}/move.txt"

                resp = @session.cmd('MOVE', @src, @dst)
                expect(resp).to be_error('DENIED')
            end
        end
    end
end