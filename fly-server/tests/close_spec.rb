RSpec.describe 'CLOSE' do
    before(:all) do
        $admin.put_array('STREAM', 'W', 'close-test.txt')
        (_, @id) = $admin.get_next

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

        (type, fr) = $admin.get_next
        expect(type).to be(:frame)
        expect(fr.id).to eq(@id)
        (type, msg) = fr.payload
        expect(type).to be(:error)
        expect(msg).to include('closed')
    end

    it 'does not create file' do
        fname = File.join($dir, 'close-test.txt')
        expect(File.exists? fname).to be false
    end

    # @TODO: test with invalid stream ID
end