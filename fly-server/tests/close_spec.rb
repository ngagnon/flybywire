RSpec.describe 'CLOSE' do
    before(:all) do
        $admin.put_array('STREAM', 'W', 'close-test.txt')
        resp = $admin.get_next
        @id = resp.value

        $admin.put_stream(@id)
        $admin.put_blob("hello1\n")

        $admin.put_array('CLOSE', Wire::Integer.new(@id))
        @line = $admin.get_string
    end

    it 'returns OK' do
        expect(@line).to eq('OK')
    end

    it 'closes stream' do
        $admin.put_stream(@id)
        $admin.put_blob("hello1\n")

        resp = $admin.get_next
        expect(resp).to be_a(Wire::Frame)
        expect(resp.id).to eq(@id)
        expect(resp.payload).to be_a(Wire::Error)
        expect(resp.payload.msg).to include('closed')
    end

    it 'does not create file' do
        fname = File.join($dir, 'close-test.txt')
        expect(File.exists? fname).to be false
    end

    # @TODO: test with invalid stream ID
end