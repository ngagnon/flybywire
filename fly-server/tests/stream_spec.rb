RSpec.describe 'STREAM' do
    context 'authorized' do
        describe 'write' do
            before(:all) do
                $admin.put_array('STREAM', 'W', 'test.txt')
                (@type, @id) = $admin.get_next
            end

            it 'returns stream id' do
                expect(@type).to be(:int)
            end

            it 'ignores frames with invalid stream ID' do
                $admin.put_stream(2)
                $admin.put_blob("hello1\n")
                (type, fr) = $admin.get_next
                expect(type).to be(:frame)
                expect(fr.id).to eq(2)
                (type, msg) = fr.payload
                expect(type).to be(:error)
                expect(msg).to include('closed')

                $admin.put_stream(1000000)
                $admin.put_blob("hello2\n")
                (type, fr) = $admin.get_next
                expect(type).to be(:frame)
                expect(fr.id).to eq(1000000)
                (type, msg) = fr.payload
                expect(type).to be(:error)
                expect(msg).to include('closed')
            end

            # @TODO: concurrent streams

            it 'writes to file' do
                $admin.put_stream(@id)
                $admin.put_blob("hello1\n")

                $admin.put_stream(@id)
                $admin.put_blob("hello2\n")

                $admin.put_stream(@id)
                $admin.put_blob("hello3\n")

                $admin.put_stream(@id)
                $admin.put_null

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
                $admin.put_array('STREAM', 'W', 'test-read.txt')
                (_, id) = $admin.get_next

                $admin.put_stream(id)
                $admin.put_blob("hello1\nhello2\nhello3\nfoobar\n")

                $admin.put_stream(id)
                $admin.put_null

                filepath = File.join($dir, 'test-read.txt')
                i = 10

                until File.exists?(filepath) || i == 0
                    sleep 0.100
                    i = i - 1
                end

                $admin.put_array('STREAM', 'R', 'test-read.txt')
                (@type, @id) = $admin.get_next
            end

            it 'returns stream id' do
                expect(@type).to be(:int)
            end

            # @TODO: concurrent streams

            it 'sends chunks' do
                (type, frame) = $admin.get_next
                expect(type).to be(:frame)
                expect(frame.id).to eq(@id)
                
                (type, payload) = frame.payload
                expect(type).to be(:blob)
                expect(payload).to eq("hello1\nhello2\nhello3\nfoobar\n")

                (type, frame) = $admin.get_next
                expect(type).to be(:frame)
                expect(frame.id).to eq(@id)
                
                (type, _) = frame.payload
                expect(type).to be(:null)
            end

            it 'returns NOTFOUND when file does not exist' do
                $admin.put_array('STREAM', 'R', 'test-not-exist.txt')
                (type, msg) = $admin.get_next
                expect(type).to be(:error)
                expect(msg).to start_with('NOTFOUND')
            end
        end
    end
end