RSpec.describe 'STREAM' do
    context 'authorized' do
        describe 'write' do
            before(:all) do
                admin.put_array('STREAM', 'W', 'test.txt')
                @resp = admin.get_next
                @id = @resp.value
            end

            it 'returns stream id' do
                expect(@resp).to be_a(Wire::Integer)
            end

            it 'ignores frames with invalid stream ID' do
                admin.put_stream(2)
                admin.put_blob("hello1\n")
                resp = admin.get_next
                expect(resp).to be_a(Wire::Frame)
                expect(resp.id).to eq(2)
                expect(resp.payload).to be_a(Wire::Error)
                expect(resp.payload.msg).to include('closed')

                admin.put_stream(1000000)
                admin.put_blob("hello2\n")
                resp = admin.get_next
                expect(resp).to be_a(Wire::Frame)
                expect(resp.id).to eq(1000000)
                expect(resp.payload).to be_a(Wire::Error)
                expect(resp.payload.msg).to include('closed')
            end

            # @TODO: concurrent streams

            it 'writes to file' do
                admin.put_stream(@id)
                admin.put_blob("hello1\n")

                admin.put_stream(@id)
                admin.put_blob("hello2\n")

                admin.put_stream(@id)
                admin.put_blob("hello3\n")

                admin.put_stream(@id)
                admin.put_null

                filepath = File.join($dir, 'test.txt')
                i = 10

                until File.exists?(filepath) || i == 0
                    sleep 0.100
                    i = i - 1
                end

                content = File.read(filepath)
                expect(content).to eq "hello1\nhello2\nhello3\n"
            end
        end

        describe 'read' do
            before(:all) do
                admin.put_array('STREAM', 'W', 'test-read.txt')
                resp = admin.get_next
                @id = resp.value

                admin.put_stream(@id)
                admin.put_blob("hello1\nhello2\nhello3\nfoobar\n")

                admin.put_stream(@id)
                admin.put_null

                filepath = File.join($dir, 'test-read.txt')
                i = 10

                until File.exists?(filepath) || i == 0
                    sleep 0.100
                    i = i - 1
                end

                admin.put_array('STREAM', 'R', 'test-read.txt')
                @resp = admin.get_next
            end

            it 'returns stream id' do
                expect(@resp).to be_a(Wire::Integer)
            end

            # @TODO: concurrent streams

            it 'sends chunks' do
                resp = admin.get_next
                expect(resp).to be_a(Wire::Frame)
                expect(resp.id).to eq(@id)
                
                payload = resp.payload
                expect(payload).to be_a(Wire::Blob)
                expect(payload.value).to eq("hello1\nhello2\nhello3\nfoobar\n")

                resp = admin.get_next
                expect(resp).to be_a(Wire::Frame)
                expect(resp.id).to eq(@id)
                expect(resp.payload).to be_a(Wire::Null)
            end

            it 'returns NOTFOUND when file does not exist' do
                admin.put_array('STREAM', 'R', 'test-not-exist.txt')
                resp = admin.get_next
                expect(resp).to be_a(Wire::Error)
                expect(resp.code).to eq('NOTFOUND')
            end
        end
    end
end